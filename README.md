## REST API 
Как я понял Service-to-service, потому что в требованиях нет OIDC ( response_type=id_token … , openid-scope, nonce и тд)  
тогда как для браузера/мобилки чаще применяют PKCE + OIDC

Поэтому я сделал clientId + Client-secret аутентификацию(для нового клиента выдаем эти реквизиты, по ним же сможем настраивать для конкретного клиента разные приколы)

В Postman

```sh
Добавьте заголовок `Authorization`
    - `Basic Auth`
    - Username: `client_id` (например, `my-auth-server`)
    - Password: ваш `client_secret` (например, `secret_key_123`)
   Postman автоматически сформирует заголовок:
        `Authorization: Basic bXktYXV0aC1zZXJ2ZXI6c2VjcmV0X2tleV8xMjM=`
```

Base URL: `http://localhost:8080/api`

| Method | Path | Description |
|--------|------|-------------|
| GET    | `/ping`                  | Health-check  
| POST   | `/auth/token/issue`      | Issue a new **access / refresh** pair  
| POST   | `/auth/token/refresh`    | Exchange an expired access token + valid refresh token for a fresh pair  
| POST   | `/auth/logout`           | Revoke **all** refresh-token sessions of a user  
| GET    | `/auth/user/guid`        | Protected endpoint – returns user ID from a valid access token  

refresh_token — это ключ к сессии
При выдаче пары токенов сервер сохраняет связь между access_token и refresh_token (например, в базе данных или в зашифрованном виде внутри самого refresh_token).
При обновлении достаточно только refresh_token


Клиент, который хочет использовать OAuth-сервер, регистрируется и получает свой `client_id` и `client_secret`.  
client_id позволяет быстро найти client_secret, а client_secret подтверждает подлинность.  
Это стандарт OAuth 2.0 (RFC 6749)

ClientID + ClientSecret и привязка по IP/UA решают разные задачи и дают разный уровень доверия.
    1. IP + User-Agent - гарантирует обращение пришло с того же браузера/мобилки, что и раньше.
        IP может смениться, UA легко подделать, заголовки могут быть потеряны

    2. ClientID + ClientSecret - Однозначно определяет приложение (back, мобилк, партнёрский сервис).
        Позволяет: выдавать разные лимиты и ttl, ставить приоритет, отзsвать доступ.