## Setup Pico temperature monitor
Pico should be connected with BOOTSEL button pressed to be in file transfer mode (it will show up in finder as mounted USB drive). Once the .uf2 file is transferred to the pico, it will reboot automatically and start running. The book would have copy to /mnt/RPI-RP2, but I found RPI-RP2 in /media/rachel.

- build picoserver
    `tinygo build -target=pico -o main.uf2 .`

- copy to pico 
    `cp main.uf2 /media/rachel/RPI-RP2`

- run tinygo monitor to get the listening address which will be something like http://192.168.0.152:80

## Start Prometheus
Navigate to picotempexport directory
    `docker compose up`

## Build picotempexport
    `docker build -t picotempexport:v1 .`

- when it's finished building:
    `docker run -d \
    --name picotempexport-v1 \
    -p 3030:3030 \
    --env PICO_SERVER_URL=http://192.168.0.152 \
    --net=picotempexport_prom_net \
    picotempexport:v1`
    