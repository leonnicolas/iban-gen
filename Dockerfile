ARG FROM=scratch
FROM $FROM
ARG GOARCH=amd64
LABEL maintainer="leonnicolas <leonloechner@gmx.de>"
COPY bin/linux/$GOARCH/iban-gen /opt/bin/
ENTRYPOINT ["/opt/bin/iban-gen"]
