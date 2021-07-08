FROM golang:alpine AS build

ARG race
ARG plugin_type=shared

RUN apk update && apk add yarn alpine-sdk upx

RUN go get -v github.com/gobuffalo/packr/v2/packr2

WORKDIR /squirssi

COPY . .

RUN go get -v ./...

RUN export SQUIRCY3_REVISION=$(cat go.mod | grep squircy3 | cut -d' ' -f2 | cut -d'-' -f3) && \
    git clone https://code.dopame.me/veonik/squircy3 ../squircy3 && \
    cd ../squircy3 && \
    git checkout $SQUIRCY3_REVISION

RUN make clean dist RACE=${race} PLUGIN_TYPE=${plugin_type}



FROM alpine:latest

RUN apk update && \
    apk add yarn alpine-sdk upx

COPY --from=build /squirssi/out/squirssi_linux_amd64 /bin/squirssi

COPY --from=build /squirssi/out/*.so /home/squirssi/.squirssi/plugins

RUN cd /home/squirssi/.squirssi/plugins && \
    for f in `ls`; do ln -sf $f `echo $f | sed -e 's/_linux_amd64//'`; done

RUN adduser -D -h /home/squirssi squirssi && \
    chown -R squirssi: /home/squirssi

USER squirssi

CMD /bin/squirssi
