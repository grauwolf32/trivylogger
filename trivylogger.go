package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	secure "github.com/cyphar/filepath-securejoin"
)

const (
	//MAXMEMORY : max filesize to upload
	MAXMEMORY = 16 * 1024 * 1024

	//BASEDIR : base directory for uploaded files
	BASEDIR = "files"
)

var (
	//InfoLogger : log info messages
	InfoLogger *log.Logger

	//ErrorLogger : Log errors
	ErrorLogger *log.Logger
)

//RequestData :
type RequestData struct {
	ProjectName string
	Branch      string
	Commit      string
	FileData    []byte
}

//exists : check if directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//InitLoggers : init loggers
func InitLoggers(logFile string) (err error) {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Llongfile)
	return
}

//SaveToFile : saves request data to file
func SaveToFile(data *RequestData) (err error) {
	if len(data.FileData) == 0 {
		return
	}

	dt := time.Now()
	if ok, err := exists(BASEDIR); err == nil {
		if !ok {
			os.Mkdir(BASEDIR, 0700)
		}
	} else {
		return err
	}

	directory := fmt.Sprintf("%s/%s", BASEDIR, dt.Format("2006-01-02"))
	if ok, err := exists(directory); err == nil {
		if !ok {
			os.Mkdir(directory, 0700)
		}
	} else {
		return err
	}

	fileName := fmt.Sprintf("%s_%s_%s.log", data.ProjectName, data.Branch, data.Commit)
	fullPath, err := secure.SecureJoin(directory, fileName)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fullPath, data.FileData, 0644)
	if err != nil {
		return err
	}

	return
}

func readFormValue(value *string, buff *[]byte, p *multipart.Part) (err error) {
	n, err := p.Read(*buff)

	if err != nil {
		if err != io.EOF {
			return err
		}
	}

	if n != 0 {
		*value = string((*buff)[:n])
	}
	return nil
}

func readFileData(result *[]byte, buff *[]byte, p *multipart.Part) (err error) {
	for {
		n, err := p.Read(*buff)
		if err != nil {
			if err == io.EOF {
				*result = append(*result, (*buff)[:n]...)
				break
			}
			return err
		}
		*result = append(*result, (*buff)[:n]...)
	}
	return nil
}

//UploadHandler : handles upload request
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	var err error

	r.Body = http.MaxBytesReader(w, r.Body, MAXMEMORY)
	reader, err := r.MultipartReader()

	InfoLogger.Printf("%s /upload %s %d\n", r.Method, r.RemoteAddr, r.ContentLength)

	if err != nil {
		ErrorLogger.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	var formData RequestData
	buff := make([]byte, 4096)

	for {
		p, err := reader.NextPart()

		if err != nil {
			if err == io.EOF {
				break
			}

			ErrorLogger.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch p.FormName() {
		case "project":
			{
				err = readFormValue(&formData.ProjectName, &buff, p)
			}

		case "branch":
			{
				err = readFormValue(&formData.Branch, &buff, p)
			}
		case "commit":
			{
				err = readFormValue(&formData.Commit, &buff, p)
			}
		case "file":
			{
				err = readFileData(&formData.FileData, &buff, p)
			}
		}

		if err != nil {
			ErrorLogger.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if formData.Commit == "" {
		errDescr := "commit should not be empty"
		http.Error(w, errDescr, http.StatusBadRequest)
		ErrorLogger.Println(err.Error())
		return
	} else if formData.Branch == "" {
		errDescr := "branch should not be empty"
		http.Error(w, errDescr, http.StatusBadRequest)
		return
	} else if formData.ProjectName == "" {
		errDescr := "project name should not be empty"
		http.Error(w, errDescr, http.StatusBadRequest)
		return
	}

	err = SaveToFile(&formData)
	if err != nil {
		ErrorLogger.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	w.Write([]byte("OK"))
	return
}

func main() {
	InitLoggers("server.log")

	http.HandleFunc("/upload", UploadHandler)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
		InfoLogger.Printf("%s /healthz %s %d\n", r.Method, r.RemoteAddr, r.ContentLength)
		return
	})

	http.ListenAndServe(":80", nil)
}
