# Kratos Project Template

## Install Kratos

```
go get -u github.com/go-kratos/kratos/cmd/kratos/v2@latest
```

## Create a service

```
# Create a template project
kratos new server

cd server
# Add a proto template
kratos proto add api/server/server.proto
# Generate the proto code
kratos proto client api/server/server.proto
# Generate the source code of service by proto file
kratos proto server api/server/server.proto -t internal/service

go generate ./...
go build -o ./bin/ ./...
./bin/server -conf ./configs
```

## Generate other auxiliary files by Makefile

```
# Download and update dependencies
make init
# Generate API swagger json files by proto file
make swagger
# Generate API validator files by proto file
make validate
# Generate all files
make all
```

## Automated Initialization (wire)

```
# install wire
go get github.com/google/wire/cmd/wire

# generate wire
cd cmd/server
wire
```

## Docker

```bash
# build
docker build -t <your-docker-image-name> .

# run
docker run --rm -p 8000:8000 -p 9000:9000 -v </path/to/your/configs>:/data/conf <your-docker-image-name>
```

https://go-kratos.dev/docs

## 项目结构

* service 只做 grpc 的入口函数，就只做（注意只做）DTO -> DO 转换，会调用 Usecase 中的某个方法
* biz 层，包含基层含义：DDD 中的 application service，domain service + domain object；Usecase 对应 application service， 负责编排多个 Domain
  Object 或者是 Domain Service，同时可以请求 Repo 捞取或者存储 Domain Object，Usecase 里不应该有复杂的逻辑， 应该下沉到 Domain Object/Service
* Domain Object ，就是领域对象，属于贫血模型，不包含持久化，持久化在application service 处理 或者在 domain service 处理
* Domain Service 和 Domain Object 什么区别，比如 转账，划给 User Domain Object 也不合适，那就抽象一个 Domain Service，
  比如叫转账服务，相当于另外一种领域对象了，这样还是Usercase 调用这个 Domain Service
* 命名的话，Usecase -> Application service，Domain -> Domain Object（比如Order），DomainService -> Domain Service（TransferService）
* 事务是在 Usecase 开（目前kratos还在设计TransactionManager，可以先data直接处理），Usecase 开事务，通常只更新一个Aggreate
* 实际工作中存在跨 Aggreate 事务更新，同步模型：类似 Seata，来更新多个 Aggreate； 异步模型：在单个 Aggregate 内部产生 Domain Event（领域事件，注意也要考虑 Repo + Message
  也要保证强一致，解耦可以考虑 Transaction Log Tailing）

## Data (数据持久化)

### database

**comment_subject**

| field | type  | remark |
| :---  | :---: | :---:  |
| id | int64 | id |
| obj_id | int64 | obj_id |
| obj_type | int64 | obj_type |
| member_id | int64 | 作者id |
| count | int32 | 一级评论数 |
| root_count | int32 | 评论数，包括已删除 |
| all_count | int32 | 评论数，包括评论的回复 |
| state | int8 | 状态 |
| create_at | time | 创建时间 |
| update_at | time | 修改时间 |

**comment_index**

| field | type  | remark |
| :---  | :---: | :---:  |
| id | int64 | id |
| obj_id | int64 | obj_id |
| obj_type | int64 | obj_type |
| member_id | int64 | 作者id |
| root | int64 | 根评论id |
| parent | int64 | 父评论id |
| parent_member_id | int64 | 父评论的作者id |
| floor | int32 | 楼层 |
| count | int32 | 回复数量 |
| like | int32 | 点赞数量 |
| hate | int32 | 点踩数量 |
| state | int8 | 状态 |
| attrs | int32 | 属性 |
| create_at | time | 创建时间 |
| update_at | time | 修改时间 |

state: 1删除  
attrs: bit, 1置顶

**comment_content**

| field | type  | remark |
| :---  | :---: | :---:  |
| comment_content | int64 | id同comment_index.id |
| ip | int64 | ip string转int64 |
| platform | int8 | web/Android/IOS |
| device | string | 具体设备名称 |
| at_member_ids | string | @的人 ,分隔 |
| message | string | 评论的内容 |
| meta | string | 元素 背景等 |
| create_at | time | 创建时间 |
| update_at | time | 更新时间 |

### redis

**subject**

field | val
--- | --- 
type | string
key | comment_subject_cache:**obj_id**:**obj_type**
value | marshal(db.comment_subject)
expire | 8h

---

**comment_sort**

field | val
--- | --- 
type | zset
key | comment_sort_cache:**obj_id**:**obj_type**
score | db.comment_index.create_at
member | db.comment_index.id
expire | 8h

---

**comment_reply_sort**

field | val
--- | --- 
type | zset
key | comment_reply_sort_cache:**comment.root**
score | db.comment_index.floor
member | db.comment_index.id
expire | 8h

---

**comment_index**

field | val
--- | --- 
type | string
key | comment_index_cache:**comment.id**
value | marshal(db.comment_index)
expire | 8h

---

**comment_content**

field | val
--- | --- 
type | string
key | comment_content_cache:**comment.id**
value | marshal(db.comment_content)
expire | 8h
