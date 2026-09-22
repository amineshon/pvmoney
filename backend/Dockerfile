FROM pvmoney-backend:latest AS certs

FROM alpine:latest
COPY --from=certs /etc/ssl/certs /etc/ssl/certs
COPY --from=certs /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Tehran
COPY server /server
EXPOSE 8080
USER nobody
CMD ["/server"]
