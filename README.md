# sendmail
Standalone drop-in replacement for sendmail with direct send

## Features

* Full sendmail replacement for direct mail without intermediate mail services
* One standalone binary without dependencies
* Optional SMTP and HTTP backends
* Possible use as a golang package

## Install

Download binaries from [release](https://github.com/n0madic/sendmail/releases) page.

Or install from source:

```
go get -u github.com/n0madic/sendmail/cmd/sendmail
```

## Help

```
Usage of sendmail:
  -f string
    	Set the envelope sender address.
  -http
    	Enable HTTP server mode.
  -httpBind string
    	TCP address to HTTP listen on. (default "localhost:8080")
  -httpToken string
    	Use authorization token to receive mail (Token: header).
  -i	When reading a message from standard input, don't treat a line with only a . character as the end of input.
  -s string
    	Specify subject on command line.
  -senderDomain value
    	Domain of the sender from which mail is allowed (otherwise all domains). Can be repeated many times.
  -smtp
    	Enable SMTP server mode.
  -smtpBind string
    	TCP or Unix address to SMTP listen on. (default "localhost:25")
  -t	Extract recipients from message headers. IGNORED (default true)
  -v	Enable verbose logging for debugging purposes.
```

## Usage

Send email like `sendmail`:

```
$ cat mail.msg | sendmail user@example.com
```

Send email like `mail/mailx`:

```
$ echo TEST | sendmail -s "Test Subject" user@example.com
```

Send via smart host:

```bash
$ export SENDMAIL_SMART_HOST=mail.server.com
$ export SENDMAIL_SMART_LOGIN=user           # Optional
$ export SENDMAIL_SMART_PASSWORD=secret      # Optional
$ cat mail.msg | sendmail user@example.com
```

Use as SMTP service:

```
$ sendmail -smtp

$ telnet localhost 25
> HELO localhost
> MAIL FROM: sender@localhost
> RCPT TO: user@example.com
> DATA
...
```

Use as HTTP service:

```
$ sendmail -http

$ curl -X POST --data-binary @mail.msg localhost:8080
```
With authorization token:
```
$ sendmail -http -httpToken werf2t34cr243

$ curl -X POST -H 'Token: werf2t34cr243' --data-binary @mail.msg localhost:8080
```

Limit the sender's domain:

```
$ sendmail -http -smtp -senderDomain example1.com -senderDomain example2.com
```

## Use as package

```go
package main

import (
    "github.com/n0madic/sendmail"
    log "github.com/sirupsen/logrus"
)

func main() {
    envelope, err := sendmail.NewEnvelope(&sendmail.Config{
        Sender:     "sender@localhost",
        Recipients: []string{"user@example.com"},
        Subject:    "Test Subject",
        Body:       []byte("TEST"),
    })
    if err != nil {
        log.Fatal(err)
    }

    errs := envelope.Send()
    for result := range errs {
        switch {
        case result.Level > sendmail.WarnLevel:
            log.Info(result.Message)
        case result.Level == sendmail.WarnLevel:
            log.Warn(result.Error)
        case result.Level < sendmail.WarnLevel:
            log.Fatal(result.Error)
        }
    }

}
```
