## Setup Pico temperature monitor
Pico should be connected with BOOTSEL button pressed to be in file transfer mode (it will show up in finder as mounted USB drive). Once the .uf2 file is transferred to the pico, it will reboot automatically and start running. The book would have copy to /mnt/RPI-RP2, but I found RPI-RP2 in /media/rachel.

- build picoserver
    `tinygo build -target=pico -o main.uf2 .`

- copy to pico 
    `cp main.uf2 /media/rachel/RPI-RP2`

- run tinygo monitor to get the listening address(this is the PICO_SERVER_URL required to run the exporter below) which will be something like http://192.168.0.152:80

## Start Prometheus
Navigate to picotempexport directory
    `docker compose up`

## Build and run picotempexport
- build the exporter image from the dockerfile
    `docker build -t picotempexport:v1 .`

- when it's finished building, run the image (setting the environment variable PICO_SERVER_URL to the url you get when running tinygo monitor and exposing port 3030 for external connection). once running, open http://localhost:3030 (or http://rachelpi:3030 if ssh).

    `docker run -d \
    --name picotempexport-v1 \
    -p 3030:3030 \
    --env PICO_SERVER_URL=http://192.168.0.152 \
    --net=exporter_prom_net \
    picotempexport:v1` 
    
- copy custom exporter config to exporter container
    `docker cp sd_picotempexporter.yml exporter-prometheus-1:/prometheus`



debug:
run docker image with a shell running in it

`docker run -it [name of image] /bin/sh`

pwd 
ls
ls -l (long - shows more info including permissions and file size)