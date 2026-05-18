# slack-proxy

将只支持 Slack 通知的应用无缝接入钉钉机器人。本服务模拟 Slack Incoming Webhook 接口，应用只需将 Slack Webhook URL 替换为本代理地址即可。

## 特性

- 支持配置多个 Slack 路径 → 钉钉 Webhook 的映射关系
- 支持钉钉机器人**加签**认证（HMAC-SHA256）
- 将 Slack `text` + `attachments` 转换为钉钉 Markdown 消息
- YAML 配置，部署简单

## 快速开始

### 二进制运行

```bash
go build -o slack-proxy .
./slack-proxy -config config.yaml
```

### Docker

```bash
# 编辑 config.yaml 填入真实 token 和 secret
docker-compose up -d
```

## 配置说明

```yaml
server:
  port: 8080          # 监听端口

routes:
  - slack_path: /hook/team-a        # 本服务监听的路径（替换 Slack URL 中的路径）
    dingtalk:
      webhook: https://oapi.dingtalk.com/robot/send?access_token=TOKEN
      secret: YOUR_SECRET           # 钉钉加签密钥（未开启加签可删除此行）
```

## 应用侧配置

将应用中的 Slack Webhook URL：
```
https://hooks.slack.com/services/xxx/yyy/zzz
```
替换为本代理地址：
```
http://<your-host>:8080/hook/team-a
```

## 钉钉加签说明

钉钉机器人开启「加签」安全设置后，填写对应的 **加签密钥**（`secret` 字段）。代理会自动计算签名并附加到请求 URL：

```
timestamp = 当前毫秒时间戳
sign = Base64(HMAC-SHA256(timestamp + "\n" + secret))
```

## 消息格式转换

| Slack 字段 | 钉钉 Markdown 输出 |
|---|---|
| `text` | 正文内容 |
| `attachments[].title` | `#### 标题` |
| `attachments[].text` | 正文 |
| `attachments[].fields` | `- **key**: value` 列表 |
| Block Kit `blocks` | 降级使用 `text` 字段 |
