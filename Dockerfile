# Mainly from chatgpt

# Use the official Golang image as the build environment
FROM golang:1.19 AS build-env

# Set the working directory to /app
WORKDIR /app

# Copy the source code into the container
COPY boggle-nogs.go .
COPY go.mod .
COPY go.sum .
COPY head.html .
COPY foot.html .

# Build the binary with flags to enable a static build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o boggle-nogs .

# Create folder for ignore files, which we'll copy over to our FROM scratch image
# since scratch doesn't have bash, can't call mkdir from within it
RUN mkdir /ignore

# Use the scratch image as a base
FROM scratch

# Copy the binary from the build environment to the new image
COPY --from=build-env /app/boggle-nogs /

# Copy head and foot (TODO: rewrite boggle-nogs to allow placing these in a volume to be user-editable)
COPY --from=build-env /app/head.html /
COPY --from=build-env /app/foot.html /

# Folder can be used to mount ignore-*.txt files
COPY --from=build-env /ignore /ignore

# Necessary files to access internet
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /etc/passwd /etc/group /etc/

# Set the command to run the binary
CMD ["/boggle-nogs", "--ignore", "/ignore"]
