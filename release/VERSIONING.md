# ShareSerial 版本号机制

## 版本格式

采用语义化版本（Semantic Versioning）：`MAJOR.MINOR.PATCH`

```
1.0.0
│ │ │
│ │ └── PATCH: Bug 修复，向后兼容
│ └──── MINOR: 新功能，向后兼容
└────── MAJOR: 重大变更，可能不兼容
```

## 版本命名规则

| 类型 | 示例 | 说明 |
|------|------|------|
| Stable Release | `1.0.0` | 正式发布版本 |
| Pre-release | `1.1.0-beta.1` | 测试版本 |
| Release Candidate | `1.1.0-rc.1` | 发布候选版本 |

## 文件命名规范

```
shareserial-{VERSION}-{PLATFORM}-{ARCH}.{EXT}

示例:
shareserial-1.0.0-linux-x86_64.tar.gz
shareserial-1.0.0-windows-x86_64.zip
shareserial-1.0.0-darwin-amd64.tar.gz
shareserial-1.0.0-darwin-arm64.tar.gz
```

## 版本升级规则

### PATCH 版本（x.x.Z）
- Bug 修复
- 文档更新
- 小优化（不影响接口）

### MINOR 版本（x.Y.z）
- 新增功能
- 新增平台支持
- 新增 CLI 命令
- 性能优化

### MAJOR 版本（X.y.z）
- 架构重构
- 接口变更
- 协议变更
- 不兼容改动

## 版本信息文件

每个 release 包含 `VERSION.json`：

```json
{
  "version": "1.0.0",
  "release_date": "2026-06-03",
  "build_number": 1,
  "commit_hash": "a3e9c0c",
  "version_scheme": "semantic"
}
```

## 版本历史

| 版本 | 日期 | 类型 | 说明 |
|------|------|------|------|
| 1.0.0 | 2026-06-03 | Stable | 首次正式发布 |