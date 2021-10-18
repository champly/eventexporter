FROM alpine:3.14

COPY bin/eventexporter .

ENTRYPOINT ["/eventexporter"]
