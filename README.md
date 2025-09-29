#databack-operator
数据备份， 基于k8s crd+operator 实现

环境说明
-开发环境 mac

## 项目初始化

基于kubebuilder 完成初始化
```bash
https://github.com/kubernetes-sigs/kubebuilder
```

编译kubebuilder
```bash
make build
```


```bash
kubebuilder init --plugins go/v4 --domain operator.molido.com --project-name databack-operator  --repo github.com/molido/databack-operator
```


创建API
```bash
kubebuilder create api --group "" --version v1beta1 --kind Databack
```

安装CRD
```bash
make install
```

打包operator
```bash
make docker-build docker-push IMG=localhost:30002/operator/databack:v1beta1
```
发布operator
```bash
make deploy IMG=localhost:30002/operator/databack:v1beta1
```

部署databack服务
```bash
kubectl apply -f config/samples/v1beat1_databack.yaml
```

##项目开发

###定义CRD
修改： /api/v1beta1/databack_types.go

```bash
make generate
```
卸载CRD
```bash
make uninstall
```
安装CRD
```bash
make install
```

实现controller/databack_controller.go 逻辑

打包operator
```bash
make docker-build IMG=localhost:30002/operator/databack:v1beta1
```

发布operator
```bash
make deploy IMG=localhost:30002/operator/databack:v1beta1
```