# imgserve

Trivial HTTP file server to transfer Virtual Machine disk images across hosts on a LAN.
Zero-install, zero-config.

## installation instructions (Fedora 28+/Centos 7.5+)

1. download the sources:
```
go get -v github.com/fromanirh/imgserve
```

2. build the server
```
cd $GOPATH/src/github.com/fromanirh/imgserve
go build -v .
```

3. copy the binary on your path. `/usr/local/bin` is the recommended destination. The server has no dependencies, so it will not pollute your system
```
# cp imgserve /usr/local/bin/
```

4. copy the configuration helper:
```
# cp imgserve.env /etc/sysconfig
```

5. install the systemd service file:
```
# cp imgserve.service /etc/systemd/system
#  systemctl enable imgserve
#  systemctl start imgserve
```

Done!
