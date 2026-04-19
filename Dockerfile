FROM scratch
COPY enetsender /app/enetsender
EXPOSE 8080
CMD ["/app/enetsender"]
