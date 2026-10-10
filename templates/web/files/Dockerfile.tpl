FROM alpine:3

RUN apk add --no-cache ca-certificates

COPY <% .Name %> /usr/local/bin/<% .Name %>

ENTRYPOINT ["<% .Name %>"]
