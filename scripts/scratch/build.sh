CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o ./scripts/scratch/tx

cat > ./scripts/scratch/Dockerfile << EOF
FROM scratch
COPY tx /
WORKDIR /workspace
ENTRYPOINT ["/tx"]
EOF

docker build -t transifex/tx ./scripts/scratch

rm ./scripts/scratch/Dockerfile ./scripts/scratch/tx

alias tx='docker run --rm -i -t -v `pwd`:/workspace -v ~/.transifexrc:/.transifexrc -v /etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt transifex/tx --root-config /.transifexrc'
