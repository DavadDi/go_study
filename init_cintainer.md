# k8s 之 Init Containers

## 1. Init Containers

在一个app pod中可以运行多个 containers， 同样可以运行多个 init containers，init container在 app container启动前运行。主要应用场景在于启动 container以前需要做初始化的动作：

* 依赖于外部启动的其他pod，Etcd或者Mysql等
* 等待下载相关依赖包后才能够继续启动
* 由于安全原因不能够将部分工具放入到 pod app container中



Init Container 在 v1.3 版本中作为 Alpha 版本引入，在 v1.6 版本前，做为 annotations 来进行使用；在 v1.6版本后，直接出现在了 spec 中。

```yaml
spec:
  containers:
  - name: myapp-container
    image: busybox
    command: ['sh', '-c', 'echo The app is running! && sleep 3600']
  initContainers:
  - name: init-myservice
    image: busybox
    command: ['sh', '-c', 'until nslookup myservice; do echo waiting for myservice; sleep 2; done;']
  - name: init-mydb
    image: busybox
    command: ['sh', '-c', 'until nslookup mydb; do echo waiting for mydb; sleep 2; done;']
```



如果 pod 一个 init container 失败了，且 pod 设置了 **restartPolicy** 策略，那么 k8s 会一直重启 pod， 直至 init container 成功运行。



init container 基本上等同于普通的 container， 包括  resource limits，volumes和security settings，但是 resource 请求略有不同。 Init contianer 由于必须要在 pod ready 前就需要完成，因此不支持 **readness probes**。 如果指定了多个 init container，那么他们会按照顺序执行，必须上一个成功执行后才会进入下一个的执行；

如果pod重启了，所有的 init container 都需要重新运行一遍；init container 的运行可能会多次检查，因此需要考虑幂等性；

使用Pod定义 `activeDeadlineSeconds` 和 Container的 `livenessProbe` 可以防止 init containers 一直重复失败. activeDeadlineSeconds 包含了 init container 启动的时间。

## 2. Init Container Example

init.yaml

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp-pod
  labels:
    app: myapp
spec:
  containers:
  - name: myapp-container
    image: busybox
    command: ['sh', '-c', 'echo The app is running! && sleep 3600']
  initContainers:
  - name: init-myservice
    image: busybox
    command: ['sh', '-c', 'until nslookup myservice; do echo waiting for myservice; sleep 2; done;']
  - name: init-mydb
    image: busybox
    command: ['sh', '-c', 'until nslookup mydb; do echo waiting for mydb; sleep 2; done;']
```

init_srv.yaml

```yaml
kind: Service
apiVersion: v1
metadata:
  name: myservice
spec:
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
---
kind: Service
apiVersion: v1
metadata:
  name: mydb
spec:
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9377
```

```shell
$ kubectl create -f init.yaml
$ kubectl get -f init.yaml
NAME        READY     STATUS    RESTARTS   AGE
myapp-pod   1/1       Running   0          5m

$ kubectl describe -f init.yaml
$ kubectl create -f init_srv.yaml
```



参考：

* https://kubernetes.io/docs/concepts/workloads/pods/init-containers/