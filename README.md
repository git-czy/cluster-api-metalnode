### [cluster-api-metalnode](https://github.com/git-czy/cluster-api-metalnode)

#### 1.简介

此项目是 cluster-api-provider-demo子项目

- cluster-api-metalnode包含metalNode CRD
- cluster-api-metalnode需要配合[cluster-api-provider-demo](https://github.com/git-czy/cluster-api-provider-demo)项目使用

- metalNode实际代表的是您的一台物理机或者虚拟机（目前只测试过centos7系统暂未做其他系统适配）
- metalNode通过ssh与您的机器通讯，并远程执行命令或者上传文件

#### 2.部署

##### 2.1.部署前准备

1. 准备一台机器安装kind，kubectl，并拉起一个集群作为manager cluster
2. 准备好装有centos7的额外至少2台机器（单master 多worker），保证22端口打开
3. 确保机器之间的网络通信正常
4. 确保已经安装clusterctl

##### 2.2.开始部署

1. 下载cluster-api-metalnode项目代码到您本地，并进入项目目录

2. 执行make run可在集群外运行项目

3. 执行make deploy将controller部署到集群

   1. 如果部署失败，请提前下载一下镜像 使用kind导入到集群

      ```
      # iamges
      ccr.ccs.tencentyun.com/oldcc/metal-node-controller:latest
      
      kind load docker-image ccr.ccs.tencentyun.com/oldcc/metal-node-controller:latest
      ```

4. 发布metalNode

   1. 修改config/samples/metal_v1beta1_metalnode.yaml，示例如下

      **注意**

      - **确保在配合 cluster-api-provider-demo项目使用时，需要将所有创建的资源使用同一Namespace**
      - **initializationCmd会覆盖默认init操作（在您的机器安装kubeadm，docker，kubelet，kubectl），如果您不熟悉kubeadm部署集群请勿启用该字段**

   ```
   apiVersion: v1
   kind: Namespace
   metadata:
     name: demo-cluster
   ---
   apiVersion: bocloud.io/v1beta1
   kind: MetalNode
   metadata:
     name: metalnode-centos-41
     namespace: demo-cluster
   spec:
     # TODO(user): Add fields here
     nodeName: centos-40
     nodeEndPoint:
       host: 10.20.30.40
       sshAuth:
         user: centos
         password: 12345678
         port: 22
   #  initializationCmd:
   #    cmds:
   #      - "echo hello world"
   
   ---
   apiVersion: bocloud.io/v1beta1
   kind: MetalNode
   metadata:
     name: metalnode-centos-41
     namespace: demo-cluster
   spec:
     # TODO(user): Add fields here
     nodeName: centos-41
     nodeEndPoint:
       host: 10.20.30.41
       sshAuth:
         user: centos
         password: Ccc51521!
         port: 22
   #  initializationCmd:
   #    cmds:
   #      - "echo hello world"
   ```

5. 查看metaNode

   ```
   kubectl get mn -n [your namespace]
   ```

   等待 Ready = true, state=SUCCESS

   ```
   NAME                   READY   STATE     ROLE   CLUSTER
   metalnode-centos-147   true    SUCCESS          
   metalnode-centos-148   true    SUCCESS  
   ```

6. 部署cluster-api-provider-demo项目

   [link](https://github.com/git-czy/cluster-api-provider-demo/blob/main/README.md)



​		

