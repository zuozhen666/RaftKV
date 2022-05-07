# RaftKV
## 代码运行
### 方式一
`./RaftKV :12100 localhost:9091 localhost:9092 localhost:9093`  
> :12100为KVServer端口号，localhost:9091为当前节点的raft通信地址，后续ip为集群里其他节点的通信地址  
### 方式二
`goreman start`
> Procfile为[goreman进程管理工具](https://github.com/mattn/goreman)的配置文件  
## 系统访问
`curl -L localhost:12100/key -XPUT -d value // 添加新的key-value pair`  
`curl -L localhost:12100/key // 获取key的value`  
`curl -L localhost:12100/key -XDELETE // 删除key`