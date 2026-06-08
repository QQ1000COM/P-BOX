# P-BOX 安装与管理密码说明

## Linux 安装

推荐安装到 `/etc/p-box`，使用 systemd 管理服务：

```bash
curl -fsSL https://raw.githubusercontent.com/star8618/P-BOX/main/install.sh | sudo bash
```

安装完成后访问：

```text
http://服务器IP:8666
```

常用命令：

```bash
systemctl status p-box
systemctl restart p-box
systemctl stop p-box
journalctl -u p-box -n 100 --no-pager
```

## 默认账号

默认管理账号：

```text
用户名：admin
密码：admin123
```

首次安装后建议进入 **设置 -> 安全设置**：

1. 开启“启用登录验证”。
2. 修改管理用户名。
3. 输入当前密码、新密码、确认密码并保存。

修改密码后，当前登录会话会失效，需要重新登录。

## 手动重置管理密码

如果忘记密码，可以在服务器上重置。下面示例把账号改为 `admin`，密码改为 `zhou.147`：

```bash
python3 - <<'PY'
import hashlib, json, pathlib

path = pathlib.Path('/etc/p-box/data/auth.json')
path.parent.mkdir(parents=True, exist_ok=True)

if path.exists():
    data = json.loads(path.read_text())
else:
    data = {'enabled': True, 'username': 'admin', 'password': '', 'avatar': ''}

password = 'zhou.147'
data['enabled'] = True
data['username'] = 'admin'
data['password'] = hashlib.sha256((password + 'p-box-salt').encode()).hexdigest()
data.setdefault('avatar', '')

path.write_text(json.dumps(data, indent=2, ensure_ascii=False))
PY

systemctl restart p-box
```

然后使用：

```text
用户名：admin
密码：zhou.147
```

## Docker 或其他容器服务访问异常

如果服务器同时运行 Docker 对外服务，建议不要开启 P-BOX 的 TUN 透明代理模式。TUN 可能接管宿主机转发流量，导致 Docker 端口映射公网访问超时。

推荐设置：

```text
透明代理模式：off
混合代理端口：7890
```

