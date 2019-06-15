FROM golang:1.12.6-stretch as build
RUN apt-get update && apt-get -uy upgrade
RUN apt-get install ca-certificates && update-ca-certificates
RUN git clone https://github.com/coredns/coredns --branch=v1.5.0 --depth=1 /go/src/github.com/coredns/coredns
WORKDIR /go/src/github.com/coredns/coredns
ADD plugin.cfg plugin.cfg
RUN make gen all

FROM debian:buster-slim
COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=build /go/src/github.com/coredns/coredns/coredns /usr/local/bin/coredns
ADD Corefile /etc/Corefile
EXPOSE 53 53/udp
ENTRYPOINT ["coredns"]
CMD ["-conf", "/etc/Corefile"]
