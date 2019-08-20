## 简介

    随着云计算和大数据技术的发展，数据量明显提升，产生了大量大文件。大文件传输过程中，存在传输速度慢，数据安全等问题。该项目为客户端与服务端进行大文件传输提供了解决方案。

## 功能

### 1、分块传输

    将大文件拆分为多个文件进行传输，有效利用带宽，提高传输速率
    -> 传输开始
    -> client获取文件信息
    -> 提供拆分方案
    -> client按照拆分方案进行拆分传输
    -> server收到拆分文件
    -> 所有拆分文件接收完成后组合成大文件
    -> 传输完成

### 2、断点续传

    提供断点续传功能，解决网络波动或者人为因素导致的传输中断问题

## 传输协议

### 1、用户登陆

client->server:上传用户名和密码

    login {user_name} {password}

### 2、上传大文件请求

client->server:上传文件名和文件大小

    big {file_name} {file_size}

server->client:返回拆分文件个数和唯一id

    {file_number} {unique_id}

### 3、上传拆分文件请求

client->server:唯一id和文件的序号（拆分的第几个文件，从0开计数）

    split {unique_id} {file_index}

server->client:续传位置

    {file_loc}

### 4、暂停上传：客户端暂停上传文件，服务端关闭连接

client->server:

    stop {unique_id} {file_index}

### 5、拆分文件上传成功，服务端校验是否所有拆分文件上传成功：如果所有文件上传成功，组装文件，并返回客户端成功标识；关闭连接

client->server:

    end {unique_id} {file_index}

server->client:

    success