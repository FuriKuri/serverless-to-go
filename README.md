# serverless-to-go

## Usage

### Start server
```
$ docker run -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock furikuri/serverless-to-go

2018/05/13 14:07:27 Listen on port 8080...
```

### Create function
*Node JS*
```
$ echo "console.log('Hello World')"  | http localhost:8080/node/hello-node
HTTP/1.1 200 OK
Content-Length: 8
Content-Type: text/plain; charset=utf-8
Date: Sat, 13 May 2018 14:19:04 GMT

FN Ready
```

*Ruby*
```
echo "puts 'Hello World'"  | http localhost:8080/ruby/hello-ruby
HTTP/1.1 200 OK
Content-Length: 8
Content-Type: text/plain; charset=utf-8
Date: Sat, 13 May 2018 14:21:43 GMT

FN Ready
```