### How to run
* Install [docker-compose][1]
* Clonse this repo, `cd` to repo folder
* Start the app by `docker-compose up db migration app`
* Run the integration test by `docker-compose up integration_tests`

### Project structure
```
.
├── README.md
├── grpc // gRPC .proto spec and generated file
│   └── server // executable server
├── integration_tests // integration test suite
├── migrations // migrations scrip use with go-migrate
├── repositories // entity definition
│   └── sql // cockroachdb implementation
├── scripts // utility script
└── services // services implementation
```

### What need to be done
* `maybe` implement generic service handler (not depend on .proto type)
* better tracing, logging support from [grpc-ecosysten][2]
* clean tests code (I wirte them in rust with lot of copy/paste)
* `HealthcheckService` to check migration state
* a build env with Dockerfile with proto compiler, [gogoslick][3]
* a 'cache' Dockerfile to share betwee, `app` and `integration_tests` for save time pulling deps
* `maybe` better to have a Docker with auto-rebuild `app`

[1]: https://docs.docker.com/compose/install/
[2]: https://github.com/grpc-ecosystem/go-grpc-middleware
[3]: https://github.com/gogo/protobuf