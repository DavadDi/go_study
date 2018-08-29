# Example Traefik for gRPC

![](https://docs.traefik.io/img/grpc.svg)

Copy exampes to $GOPATH/src/google.golang.org/grpc/examples.

1. Gen certs

    ```bash
    $ gen_certs.sh
    ```

1. Download traefik

    ```bash
    $ cd traefik
    $ wget https://github.com/containous/traefik/releases/download/v1.6.6/traefik_linux-amd64 -o traefik
    ```

1. Start Greeter Server

    ```bash
    $ cd hello_world
    $ ./start_server.sh
    ```

1. Start traefik Server

    ```bash
    $ cd traefik
    $ ./start.sh
    ```

1. Start Greeter Client

    ```bash
    $ cd traefik
    $ ./start_client.sh
    ```
