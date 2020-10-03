FROM veonik/squirssi:build-amd64 AS build


FROM debian:buster-slim

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y ca-certificates curl gnupg && \
    curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add - && \
    echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list && \
    apt-get update && \
    apt-get install -y yarn

COPY config.toml.dist /home/squirssi/.squirssi/config.toml

COPY package.json /home/squirssi/.squirssi/scripts/package.json

RUN cd /home/squirssi/.squirssi/scripts && \
    yarn install

COPY --from=build /squirssi/out/squirssi_linux_amd64 /bin/squirssi

COPY --from=build /squirssi/out/*.so /squirssi/plugins/

RUN cd /squirssi/plugins && \
    for f in `ls`; do ln -sf $f `echo $f | sed -e 's/_linux_amd64//'`; done

RUN useradd -d /home/squirssi squirssi && \
    chown -R squirssi: /home/squirssi /squirssi

USER squirssi

WORKDIR /squirssi

CMD /bin/squirssi
