#!/bin/bash
go build -o podWatch main.go
./podWatch -kubeconfig=./config-minikube
