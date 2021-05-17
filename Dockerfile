FROM golang:latest
LABEL  author="grauwolf32@mail.ru"
COPY . /build/
RUN  cd /build/ && go get github.com/cyphar/filepath-securejoin &&\
     go build /build/trivylogger.go && mkdir /app/ && \
     mkdir /app/files/ && cp /build/trivylogger /app/trivylogger

WORKDIR "/app"
ENTRYPOINT "/app/trivylogger"