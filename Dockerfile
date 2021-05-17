FROM golang:latest
LABEL  author="grauwolf32@mail.ru"
COPY . /build/
RUN go get github.com/cyphar/filepath-securejoin && go build /build/trivylogger.go \
&& cp /build/trivylogger /app/trivylogger

ENTRYPOINT ["/app/trivylogger"]