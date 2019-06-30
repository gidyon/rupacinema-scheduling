FROM scratch
# Working directory
WORKDIR /

# server.bin is app binary, certs needed for TLS
COPY server.bin /
COPY certs /certs

# Entry, no flag to show it is running in flag
ENTRYPOINT [ "/server.bin" ]