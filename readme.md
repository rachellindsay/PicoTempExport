# PicoTempExport: connect to pico-w and get data back.

Based on the book [Automate Your Home Using Go](https://pragprog.com/book/gohome) by Ricardo Gerardi and Mike Riley 

A sensor (currently temperature sensor built into the pico, but eventually plant monitor, temperature monitor, switch monitor, etc) is connected to a pico-w (pico w/ wi-fi). Code on that pico returns sensor data via http.

A raspberry pi running prometheus harvests the data and displays it.

picoserver has the code that runs on the pico_w to get the data from the sensor. This code listens for http connections (over the wifi), and returns json with information from the sensor.

exporter has the code that runs on the pi. It uses Prometheus and a custom go app to extract data from picoserver.

## picoserver

I build the code that runs on the pico-w on the pi, and then load the compiled code onto to pico-w:

1. Install tinygo on the pi: https://tinygo.org/
2. Setup the cyw43439 driver. This is the go driver that allows the pico-w to connect to wifi. Need this separate package because it is not included in tinygo pico support. cyw43439 is the wifi chip on the pico-w:
    - clone the repo from https://github.com/soypat/cyw43439 creating a directory cyw43439 at the same level as picoserver and exporter
    - follow the instructions in cyw43439/examples/common/secrets.go.template to create cyw43439/examples/common/secrets.go with wifi credentials
3. Build the main.uf2 compiled code (in the picoserver directory): `tinygo build -target=pico -o main.uf2 .`
4. Transfering code to the pico:
    - Connect the pico to the pi with the BOOTSEL button pressed. This puts the pico in file transfer mode (it will show up on the pi as mounted USB drive). The book claims it will show up as /mnt/RPI-RP2, but I found RPI-RP2 in /media/rachel. 
    - copy to pico: `cp main.uf2 /media/rachel/RPI-RP2`
    - Once the .uf2 file is transferred to the pico, it will reboot automatically and start running
5. Immediately run tinygo monitor to get the listening address which will be something like http://192.168.0.152:80. This will also let you know if the connection to wifi failed. If you don't connect to monitor pretty quickly, you could miss the connection status and listening address (it will show something like "waiting for connection"). disconnect the pico and reconnect to restart the app running on the pico. run tinygo monitor again and you will get the setup info again.
6. You should be able to see the sensor results at that ip address with any browser.

## exporter

### Start Prometheus
Navigate to exporter directory
    `docker compose up`

### Build and run picotempexport
The docker build: `docker build -t picotempexport:v1 .`

when it's finished building, run the image (setting the environment variable PICO_SERVER_URL to the url you get when running tinygo monitor ad exposing port 3030 for external connection). Once running, open http://localhost:3030 (or http://rachelpi:3030 if ssh).
```
    docker run -d \
      --name picotempexport-v1 \
      -p 3030:3030 \
      --env PICO_SERVER_URL=http://192.168.0.152 \
      --net=exporter_prom_net \
      picotempexport:v1
```

- when it's finished building, run the image (setting the environment variable PICO_SERVER_URL to the url you get when running tinygo monitor and exposing port 3030 for external connection). once running, open http://localhost:3030 (or http://rachelpi:3030 if ssh).

    `docker run -d \
    --name picotempexport-v1 \
    -p 3030:3030 \
    --env PICO_SERVER_URL=http://192.168.0.152 \
    --net=exporter_prom_net \
    picotempexport:v1` 
    
copy custom exporter config to exporter container: `docker cp sd_picotempexporter.yml exporter-prometheus-1:/prometheus`

## debug hints:
run docker image with shell running in it: `docker run -it [name of image] /bin/sh`

useful commands:
- ls -l (long, shows more info including permissions and file size)
- pwd (make sure you are where you think you are)