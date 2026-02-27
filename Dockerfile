FROM golang:1.25.7
WORKDIR /app
COPY . ./
RUN CGO_ENABLED=1 go build -mod=vendor -o main ./main.go

EXPOSE 8080
CMD ["./main"] 
