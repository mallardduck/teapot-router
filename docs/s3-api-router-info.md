# S3 API Router Info

This is a list of the S3 API actions and how they could be mapped to the router.

| Scope     | S3 Action Name    | Verb   | Path            | Query Param             | Description                                      |
|-----------|-------------------|--------|-----------------|-------------------------|--------------------------------------------------|
| Service   | ListBuckets       | GET    | /               | —                       | Returns XML list of all buckets.                 |
| Bucket    | CreateBucket      | PUT    | /{bucket}       | —                       | Create bucket; check LocationConstraint in body. |
| Bucket    | DeleteBucket      | DELETE | /{bucket}       | —                       | Delete bucket; must be empty.                    |
| Bucket    | HeadBucket        | HEAD   | /{bucket}       | —                       | Fast check for existence/permissions.            |
| Bucket    | ListObjectsV2     | GET    | /{bucket}       | list-type=2             | The standard for fetching object lists.          |
| Bucket    | GetBucketLocation | GET    | /{bucket}       | location                | Returns the region (e.g., us-east-1).            |
| Bucket    | PutBucketAcl      | PUT    | /{bucket}       | acl                     | Sets canned ACLs or XML-defined policies.        |
| Object    | PutObject         | PUT    | /{bucket}/{key} | —                       | Standard upload.                                 |
| Object    | GetObject         | GET    | /{bucket}/{key} | —                       | Download object; handles Range headers.          |
| Object    | DeleteObject      | DELETE | /{bucket}/{key} | —                       | Remove single object.                            |
| Object    | HeadObject        | HEAD   | /{bucket}/{key} | —                       | Get metadata (size, content-type) only.          |
| Object    | CopyObject        | PUT    | /{bucket}/{key} | —                       | Triggered by x-amz-copy-source header.           |
| Multipart | CreateMultipart   | POST   | /{bucket}/{key} | uploads                 | Returns an UploadId for subsequent parts.        |
| Multipart | UploadPart        | PUT    | /{bucket}/{key} | partNumber=X&uploadId=Y | Upload specific chunk.                           |
| Multipart | CompleteMultipart | POST   | /{bucket}/{key} | uploadId=Y              | Finalizes upload; triggers part merging.         |
| Multipart | AbortMultipart    | DELETE | /{bucket}/{key} | uploadId=Y              | Cleanup of uncommitted parts.                    |
| Multipart | ListParts         | GET    | /{bucket}/{key} | uploadId=Y              | List parts uploaded so far for this ID.          |