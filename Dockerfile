FROM scratch
MAINTAINER Yann HAMON <yann@mandragor.org>
ADD redis-dump-go /
ENTRYPOINT ["/redis-dump-go"]
