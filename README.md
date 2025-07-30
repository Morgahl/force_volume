# Force Volume

A simple tool to force the input volume of a device to a specific value and maintain it there on an interval.

I made it because I wanted to enforce the volume of my microphone on my laptop in Windows 11 and other programs keep
trying to adjust it out from under me.

## Usage

```sh
# Force the volume to 95% every 3 seconds
go run main.go

# Force the volume to 50% every 5 seconds
go run main.go --volume 50 --interval 5s
```

