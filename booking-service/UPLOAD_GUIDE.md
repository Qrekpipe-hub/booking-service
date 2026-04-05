# Инструкция: загрузить проект на GitHub

## Шаг 1 — Создать репозиторий на GitHub

1. Зайдите на [github.com](https://github.com) и войдите в аккаунт **Qrekpipe-hub**
2. Нажмите **+** → **New repository**
3. Заполните:
   - **Repository name:** `booking-service`
   - **Description:** `REST-сервис бронирования переговорок` (опционально)
   - **Visibility:** Public или Private — на ваш выбор
   - ❌ НЕ ставьте галочки «Add README», «Add .gitignore» — репозиторий должен быть пустым
4. Нажмите **Create repository**

---

## Шаг 2 — Распаковать архив

```bash
unzip booking-service.zip
cd booking-service
```

---

## Шаг 3 — Инициализировать Git и сделать первый коммит

```bash
git init
git add .
git commit -m "init"
```

---

## Шаг 4 — Привязать к репозиторию на GitHub и запушить

```bash
git remote add origin https://github.com/Qrekpipe-hub/booking-service.git
git branch -M main
git push -u origin main
```

> Если GitHub попросит логин — введите username **Qrekpipe-hub** и **Personal Access Token** вместо пароля.
> Создать токен: GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic) → Generate new token → поставить галочку **repo** → Generate.

---

## Шаг 5 — Проверить

Откройте `https://github.com/Qrekpipe-hub/booking-service` — все файлы должны быть на месте.

---

## Если уже есть SSH-ключ (альтернатива токену)

```bash
git remote add origin git@github.com:Qrekpipe-hub/booking-service.git
git branch -M main
git push -u origin main
```

---

## Итоговые команды одним блоком

```bash
unzip booking-service.zip
cd booking-service
git init
git add .
git commit -m "init"
git remote add origin https://github.com/Qrekpipe-hub/booking-service.git
git branch -M main
git push -u origin main
```
