# RaftKV
## 当前进度
- [x] 实现一个简易支持GET、PUT、DELETE的kv服务
- [x] 实现raft模块算法
- [x] 设计raft与kvServer结合方案
- [x] 实现多kvServer管理
- [ ] 整合完善系统

## ToDo
- [x] 新leader节点被选举出来后的log replication bug
- [x] 新加入节点的log replication bug
- [X] 测试一下5个节点的cluster行为
- [ ] kv server到raft module的同步问题
- [ ] 对leader添加新的日志类型-广播集群新成员添加