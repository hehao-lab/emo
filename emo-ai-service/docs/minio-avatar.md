# MinIO avatar storage

Avatar uploads use `POST /v1/files/avatar` with an authenticated
`multipart/form-data` field named `file`. The service accepts JPEG, PNG, WebP,
and GIF files up to 5 MB, stores them in MinIO under the authenticated user's
prefix, and returns a `publicUrl`. The client then saves that URL through the
existing `PUT /v1/users/me/avatar` endpoint.

The backend creates the `emotion-avatars` bucket on the first upload and sets
an anonymous download policy. Use a dedicated bucket because everything in it
is intended to be publicly readable avatar media.

## Configuration

Set these environment variables before starting the backend. Do not put the
secret key in `configs/config.yaml` or commit it to source control.

```powershell
$env:EMO_MINIO_ENDPOINT = "http://127.0.0.1:9000"
$env:EMO_MINIO_PUBLIC_ENDPOINT = "http://127.0.0.1:9000"
$env:EMO_MINIO_ACCESS_KEY = "<your-minio-access-key>"
$env:EMO_MINIO_SECRET_KEY = "<your-minio-secret-key>"
$env:EMO_MINIO_BUCKET = "emotion-avatars"
$env:EMO_MINIO_REGION = "us-east-1"
```

`EMO_MINIO_ENDPOINT` is the address the backend uses to reach MinIO.
`EMO_MINIO_PUBLIC_ENDPOINT` is the address embedded in returned avatar URLs,
so it must be reachable by the app. When both services run on the host and
MinIO publishes port `9000`, both values can be `http://127.0.0.1:9000`.
When the backend runs in Docker, the internal endpoint can be
`http://minio:9000`, while the public endpoint should be the externally
reachable domain or host address.

## Client flow

1. Select an image in the settings page.
2. Upload it with `uni.uploadFile` to `/v1/files/avatar` using the JWT bearer
   token and the multipart field name `file`.
3. Send the returned `publicUrl` to `PUT /v1/users/me/avatar`.

The profile endpoint validates that the URL belongs to the authenticated user's
avatar prefix in the configured public MinIO endpoint. It rejects external
URLs and URLs belonging to another user.

The mobile settings page implements this sequence in
`common/profile-api.mjs` and `components/settings/SettingsScreen.vue`.
