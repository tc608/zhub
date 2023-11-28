# ZHub 快速上手

[ZHub 快速上手：https://docs.1216.top](https://docs.1216.top)

## 概述
> zhub是⼀个⾼性能事件发布订阅服务组件，功能丰富，包含发布-订阅、⼴播消息、延时消息、
Rpc调⽤、分布式定时调度、分布式锁，运⾏包仅有1M+；低服务资源消费，初始启动内存 10M-。

![zhub-fun.png](https://img.1216.top/docs/zhub/zhub-fun.png)

---

## 开始 搭建 zhub 服务
> 让我们在 **5到10分钟内完成 zhub 中间件安装、集成、测试**.

### 下载软件包

- [zhub.zip (点击下载)](https://img.1216.top/docs/zhub/zhub.zip) 包含以下内容:
    - `zhub-client-0.1.1.dev.jar`  常规Java 项目驱动包
    - `zhub-client-spring-0.1.1.jar` springboot 项目驱动包
    - `zhub.exe` Window 运行包
    - `zhub.sh` Linux 运行包
    - `zhub`    Mac 运行包
    - `app.ini` 配置文件
      ![zhub-zip.png](https://img.1216.top/docs/zhub/dist-zip.png)

### 配置 app.ini

```bash
# app.ini
[service]
watch=0.0.0.0:711           # 服务管理端口
addr=0.0.0.0:1216           # 服务端口
auth=0                      # 是否开启连接授权 0不开启、1开启

[data]
dir=./data                  # 数据目录

[log]
handlers=console            # console|file
level=debug                 # info|debug|error
file=zhub.log

[ztimer]                    # ztimer 配置 (可选，如果不使用定时调度则可不配置)
# db.addr=127.0.0.1:3306    # timer 使用的MySql数据库配置
# db.user=root
# db.password=123456
# db.database=zhub
```

---

### 初始化 ztimer 数据库 (可选，如果不使用定时调度则不需要配置)

```sql
CREATE DATABASE zhub;
CREATE TABLE `zhub`.`tasktimer` (
  `timerid` varchar(64) NOT NULL DEFAULT '' COMMENT '[主键]UUID',
  `name` varchar(32) NOT NULL DEFAULT '' COMMENT '[任务名称]',
  `expr` varchar(32) NOT NULL DEFAULT '' COMMENT '[时间表达式]',
  `single` int NOT NULL DEFAULT '1' COMMENT '[单实例消费]1单对象，0不限',
  `remark` varchar(128) NOT NULL DEFAULT '' COMMENT '[备注]',
  `status` smallint NOT NULL DEFAULT '10' COMMENT '[状态]10启用,60停用',
  PRIMARY KEY (`timerid`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
# 初始化 四个定时任务配置, 订阅名称分别为 T:A、T:B、T:C、T:D 
INSERT INTO `zhub`.`tasktimer` (`timerid`, `name`, `expr`, `single`, `remark`, `status`) VALUES 
  ('T1', 'T:A', '*/5 * * * * ?', 1, '每5秒执行一次', 10),
  ('T2', 'T:B', '15s', 1, '每15秒执行一次', 10),
  ('T3', 'T:C', '0 0 0 * * 1', 0, '每周一00:00执行', 10),
  ('T4', 'T:D', '0 0 24 * * ?', 1, '每天00:00执行', 10);
```

### 启动服务

```bash
# window 
./zhub.exe
# linux (添加执行权限 chmod +x ./zhub.sh)
./zhub.sh
```
---
### 使用 zhub 收发消息

 [使用 zhub 收发消息 https://docs.1216.top](https://docs.1216.top)
