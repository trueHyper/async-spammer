# Async Spam Checker

Имитирует Unix-пайплайн для проверки писем на спам:
```bash
cat emails.txt | SelectUsers | SelectMessages | CheckSpam | CombineResults
```
## Пайплайн

### SelectUsers
- **in:** `string` (имейлы пользователей)  
- **out:** `User{}` (результат `GetUser()`)  
- **особенности:**  
  - `GetUser()` занимает ~1 секунду, можно вызывать параллельно  
  - Убирает дубликаты, учитывая alias'ы пользователей  

### SelectMessages
- **in:** `User{}`  
- **out:** `MsgID` (результат `GetMessages()`)  
- **особенности:**  
  - `GetMessages()` занимает ~1 секунду  
  - Поддерживает батчи до 2 пользователей за раз  
  - Оптимизирует количество вызовов  

### CheckSpam
- **in:** `MsgID`  
- **out:** `MsgData{ id, has_spam }` (результат `HasSpam()`)  
- **особенности:**  
  - Один запрос ~100ms  
  - Максимум 5 параллельных вызовов, иначе сервис возвращает ошибку  

### CombineResults
- **in:** `MsgData`  
- **out:** `string` вида `<has_spam> <msg_id>`  
- Пример: `true 17696166526272393238`
