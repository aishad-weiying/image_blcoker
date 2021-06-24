from ubuntu:latest

workdir /
add main /
run chmod +x main
ENTRYPOINT ["./main"]
EXPOSE 9527
