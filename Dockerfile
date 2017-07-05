FROM golang
ADD . /go/src/github.com/tobyjsullivan/event-log
RUN  go install github.com/tobyjsullivan/event-log

EXPOSE 3000

CMD /go/bin/event-log
