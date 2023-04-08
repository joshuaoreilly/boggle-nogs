# Use the official Golang image as the build environment
FROM golang:1.19 AS build-env

# Set the working directory to /app
WORKDIR /app

# Copy the source code into the container
COPY boggle-nogs.go .
COPY go.mod .
COPY go.sum .

# Build the binary
# CGO_ENABLED for static binary
# GOOS for only a linux binary
RUN CGO_ENABLED=0 GOOS=linux go build -o boggle-nogs .

# Use alpine for final image
FROM alpine:3.17

EXPOSE 8080

WORKDIR /app

# Copy the binary from the build environment to the new image
COPY --from=build-env /app/boggle-nogs .

# Copy head and foot (TODO: rewrite boggle-nogs to allow placing these in a volume to be user-editable)
COPY head.html .
COPY foot.html .

# Folder can be used to mount ignore-*.txt files
RUN mkdir ./ignore

# Create non-root user, set as owner of relevant folders
RUN adduser -D bogglenogs
RUN chown -R bogglenogs .

# switch to non-root user
USER bogglenogs

# Set the command to run the binary
CMD ["./boggle-nogs", "--ignore", "ignore"]
