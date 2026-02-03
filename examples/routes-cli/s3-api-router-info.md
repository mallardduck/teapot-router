# S3 API Router Info

This is a list of the S3 API actions and how they could be mapped to the `teapot-rotuer`.

## S3 API Routes and Actions

| Scope     | S3 Action Name                  | Verb   | Path            | Query Param             | Description                                            |
|-----------|---------------------------------|--------|-----------------|-------------------------|--------------------------------------------------------|
| Service   | ListBuckets                     | GET    | /               | —                       | Lists all buckets; the "entry point" for most clients. |
| Bucket    | CreateBucket                    | PUT    | /{bucket}       | —                       | Creates a new bucket (handle LocationConstraint).      |
| Bucket    | DeleteBucket                    | DELETE | /{bucket}       | —                       | Deletes a bucket (must be empty).                      |
| Bucket    | HeadBucket                      | HEAD   | /{bucket}       | —                       | Checks if a bucket exists and you have access.         |
| Bucket    | ListObjectsV2                   | GET    | /{bucket}       | list-type=2             | Critical. Modern method for listing objects.           |
| Bucket    | ListObjects                     | GET    | /{bucket}       | —                       | Legacy listing (v1). Still required for old SDKs.      |
| Bucket    | GetBucketLocation               | GET    | /{bucket}       | location                | Returns the region string (e.g., us-east-1).           |
| Bucket    | GetBucketVersioning             | GET    | /{bucket}       | versioning              | Returns Enabled or Suspended.                          |
| Bucket    | PutBucketVersioning             | PUT    | /{bucket}       | versioning              | Enables/disables versioning for the bucket.            |
| Bucket    | GetBucketAcl                    | GET    | /{bucket}       | acl                     | Returns bucket-level permissions.                      |
| Bucket    | PutBucketAcl                    | PUT    | /{bucket}       | acl                     | Sets bucket-level permissions.                         |
| Bucket    | GetBucketPolicy                 | GET    | /{bucket}       | policy                  | Returns the JSON bucket policy.                        |
| Bucket    | PutBucketPolicy                 | PUT    | /{bucket}       | policy                  | Sets the JSON bucket policy.                           |
| Bucket    | DeleteBucketPolicy              | DELETE | /{bucket}       | policy                  | Removes the bucket policy.                             |
| Bucket    | GetBucketCors                   | GET    | /{bucket}       | cors                    | Critical for browser-based (JS) uploads.               |
| Bucket    | PutBucketCors                   | PUT    | /{bucket}       | cors                    | Sets Cross-Origin Resource Sharing rules.              |
| Bucket    | GetBucketLifecycle              | GET    | /{bucket}       | lifecycle               | Returns expiration/transition rules.                   |
| Bucket    | PutBucketLifecycle              | PUT    | /{bucket}       | lifecycle               | Sets automatic data deletion/transition.               |
| Bucket    | PutPublicAccessBlock            | PUT    | /{bucket}       | publicAccessBlock       | New standard. Restricts public access.                 |
| Object    | PutObject                       | PUT    | /{bucket}/{key} | —                       | Core. Uploads a single object (max 5GB).               |
| Object    | GetObject                       | GET    | /{bucket}/{key} | —                       | Core. Downloads object (supports Range).               |
| Object    | HeadObject                      | HEAD   | /{bucket}/{key} | —                       | Core. Gets size/type without downloading data.         |
| Object    | DeleteObject                    | DELETE | /{bucket}/{key} | —                       | Core. Removes a single object.                         |
| Object    | DeleteObjects                   | POST   | /{bucket}       | delete                  | Critical. Bulk delete (up to 1,000 objects).           |
| Object    | CopyObject                      | PUT    | /{bucket}/{key} | —                       | Server-side copy using x-amz-copy-source.              |
| Object    | GetObjectTagging                | GET    | /{bucket}/{key} | tagging                 | Returns key-value tags.                                |
| Object    | PutObjectTagging                | PUT    | /{bucket}/{key} | tagging                 | Sets key-value tags.                                   |
| Object    | GetObjectLegalHold              | GET    | /{bucket}/{key} | legal-hold              | Compliance: Checks if object is under legal hold.      |
| Object    | PutObjectRetention              | PUT    | /{bucket}/{key} | retention               | Compliance: Sets WORM/Immutability date.               |
| Multipart | CreateMultipartUpload           | POST   | /{bucket}/{key} | uploads                 | Starts the process for files >5GB.                     |
| Multipart | UploadPart                      | PUT    | /{bucket}/{key} | partNumber=X&uploadId=Y | Uploads a specific chunk of a large file.              |
| Multipart | UploadPartCopy                  | PUT    | /{bucket}/{key} | partNumber=X&uploadId=Y | Copies an existing object part as a new part.          |
| Multipart | CompleteMultipartUpload         | POST   | /{bucket}/{key} | uploadId=Y              | Merges all parts into a final object.                  |
| Multipart | AbortMultipartUpload            | DELETE | /{bucket}/{key} | uploadId=Y              | Stops upload and cleans up temp storage.               |
| Multipart | ListParts                       | GET    | /{bucket}/{key} | uploadId=Y              | Lists parts uploaded for an active ID.                 |
| Multipart | ListMultipartUploads            | GET    | /{bucket}       | uploads                 | Lists all unfinished multipart uploads.                |
| Security  | PutPublicAccessBlock            | PUT    | /{bucket}       | publicAccessBlock       | Blocks all public access to a bucket.                  |
| Security  | GetPublicAccessBlock            | GET    | /{bucket}       | publicAccessBlock       | Retrieves the current block settings.                  |
| Locking   | PutObjectLockConfiguration      | PUT    | /{bucket}       | object-lock             | Enables WORM (Write Once Read Many).                   |
| Locking   | GetObjectRetention              | GET    | /{bucket}/{key} | retention               | Gets retention date (governance/compliance).           |
| Locking   | PutObjectRetention              | PUT    | /{bucket}/{key} | retention               | Sets how long an object is immutable.                  |
| Locking   | PutObjectLegalHold              | PUT    | /{bucket}/{key} | legal-hold              | Places an indefinite hold on an object.                |
| Lifecycle | PutBucketLifecycleConfiguration | PUT    | /{bucket}       | lifecycle               | Auto-delete or transition old data.                    |
| Lifecycle | GetBucketLifecycleConfiguration | GET    | /{bucket}       | lifecycle               | Returns the lifecycle rule set.                        |
| Logging   | PutBucketLogging                | PUT    | /{bucket}       | logging                 | Sets where server access logs are sent.                |
| Events    | PutBucketNotification           | PUT    | /{bucket}       | notification            | Triggers events (Lambda/SQS) on upload.                |
| Payment   | GetBucketRequestPayment         | GET    | /{bucket}       | requestPayment          | Required by many SDKs to check "Requester Pays."       |
| Integrity | GetObjectAttributes             | GET    | /{bucket}/{key} | attributes              | Returns ETag, size, and storage class.                 |
| Archives  | GetObjectTorrent                | GET    | /{bucket}/{key} | torrent                 | Returns Bencoded torrent file (Legacy).                |
| Analysis  | GetBucketAnalyticsConfiguration | GET    | /{bucket}       | analytics               | Analyzes storage usage patterns.                       |


### S3 API Operations Implementation Tier

**Tier 1 (Must Have - Core Functionality):**

* All Service operations (ListBuckets)
* Core Bucket ops (CreateBucket, DeleteBucket, HeadBucket, ListObjectsV2, GetBucketLocation)
* Core Object ops (PutObject, GetObject, DeleteObject, HeadObject, CopyObject)
* All Multipart ops (CreateMultipartUpload, UploadPart, CompleteMultipartUpload, AbortMultipartUpload, ListParts, ListMultipartUploads)
*
**Tier 2 (Highly Recommended - Essential for Production):**

* DeleteObjects (bulk delete efficiency)
* ListObjects (legacy v1 support for older SDKs)
* GetBucketVersioning, PutBucketVersioning
* GetBucketCors, PutBucketCors (critical for browser-based uploads)
* GetObjectTagging, PutObjectTagging
* UploadPartCopy

**Tier 3 (Important - Security & Compliance):**

* GetBucketAcl, PutBucketAcl
* GetBucketPolicy, PutBucketPolicy, DeleteBucketPolicy
* PutPublicAccessBlock, GetPublicAccessBlock
* GetBucketRequestPayment (SDK compatibility)

**Tier 4 (Standard Features - Lifecycle & Automation):**

* GetBucketLifecycle, PutBucketLifecycle
* GetBucketLifecycleConfiguration, PutBucketLifecycleConfiguration
* PutBucketNotification
* PutBucketLogging
* GetObjectAttributes

**Tier 5 (Advanced - Compliance & Locking):**

* PutObjectLockConfiguration
* GetObjectRetention, PutObjectRetention
* GetObjectLegalHold, PutObjectLegalHold

**Tier 6 (Optional - Analytics & Legacy):**

* GetBucketAnalyticsConfiguration
* GetObjectTorrent (legacy feature, rarely used)