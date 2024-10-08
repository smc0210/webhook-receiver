# Webhook receiver

## 개요

이 애플리케이션은 웹훅 이벤트를 수신하기 위한 간단한 서버를 제공하며, 실시간으로 웹 인터페이스에 표시합니다. 또한 ngrok과 통합하여 공개 URL을 제공하므로, 로컬에서 웹훅 테스트를 손쉽게 진행할 수 있습니다.

## 기능

- 지정된 포트에서 웹훅 이벤트 수신
- 실시간으로 웹 인터페이스에 이벤트 표시
- ngrok을 통한 웹훅 리시버의 공개 URL 노출
- 공개 URL을 클립보드에 자동 복사

## 사용 방법

1. 서버 구동:
    ```
    cd corewebhook
    go run .
    ```

2. 웹훅 리시버 구동:
    ```
    cd webhook
    ./webhook-receiver
    ```

3. 로컬에서는 웹훅을 수신할 수 없으므로 ngrok을 통해 공개 URL을 얻습니다:
    ```
    ngrok http 8080
    ```

얻어진 `Forwarding` 주소를 사용하여 웹훅을 전송합니다.

## 빌드 방법

바이너리를 빌드하기 위해 다음 명령어를 실행합니다:

```bash
go build -o webhook-receiver
```

### TODO

- [ ] ngrok static domain guide
- [ ] Other case ( 5xx-200, 5xx only)
- [ ] Support x-www-form-urlencoded
