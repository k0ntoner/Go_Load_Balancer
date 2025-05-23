<img width="500" alt="image" src="https://github.com/user-attachments/assets/2d34494e-30d2-4153-a2fb-a97b01d7198e" /># Go Load Balancer — Least Connections with Auto Scaling

Цей репозиторій містить реалізацію високопродуктивного балансувальника навантаження на мові Go. Балансувальник використовує алгоритм **Least Connections** та підтримує **динамічне оновлення пулу серверів** через AWS Auto Scaling Group. Реалізовано механізми health-check, повторних спроб, логування та graceful shutdown.

---

## ⚙️ Архітектура
<img width="500" alt="image" src="https://github.com/user-attachments/assets/0ddda14c-cd7f-4d99-a297-8a9c0dba364b" />

---

## 📦 Основні компоненти

- **Dispatcher** — обирає найменш завантажений вузол згідно з алгоритмом Least Connections.
- **Worker** — відповідає за надсилання HTTP-запитів до одного конкретного серверного вузла.
- **Auto Scaling Updater** — періодично оновлює список серверів через AWS Auto Scaling Group.
- **Config Loader** — зчитує параметри запуску з конфігураційного файлу `application_properties.yaml`.

---

## 📁 Конфігурація (`application_properties.yaml`)

```yaml
port: 8080
autoScalingGroupName: my-auto-scaling-group
awsRegion: eu-north-1
refreshIntervalSec: 30
retryCount: 5
