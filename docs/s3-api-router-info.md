# S3 API Router Info

This is a list of the S3 API actions and how they could be mapped to the router.
I'll verify your table against the AWS S3 API documentation and identify missing critical routes.

## Verification of Your Table

Let me check the accuracy of each entry:Now let me check a few key operations to verify your table accuracy:Based on my research of AWS S3 API documentation, here's my comprehensive verification and updated table:

## Verification Results

Your table is **mostly accurate**, with a few corrections needed:

### Issues Found:

1. **ListMultipartUploads** - Your table shows it as listing parts for a specific upload, but it actually lists **all** incomplete multipart uploads in a bucket (query param is `uploads`, not `uploadId=Y`)
2. **Missing critical operations** - Several must-implement operations are absent

### Corrected and Enhanced Table

| Scope     | S3 Action Name          | Verb   | Path            | Query Param             | Description                                      |
|-----------|-------------------------|--------|-----------------|-------------------------|--------------------------------------------------|
| Service   | ListBuckets             | GET    | /               | —                       | Returns XML list of all buckets.                 |
| Bucket    | CreateBucket            | PUT    | /{bucket}       | —                       | Create bucket; check LocationConstraint in body. |
| Bucket    | DeleteBucket            | DELETE | /{bucket}       | —                       | Delete bucket; must be empty.                    |
| Bucket    | HeadBucket              | HEAD   | /{bucket}       | —                       | Fast check for existence/permissions.            |
| Bucket    | ListObjects             | GET    | /{bucket}       | —                       | List objects (v1, prefer ListObjectsV2).          |
| Bucket    | ListObjectsV2           | GET    | /{bucket}       | list-type=2             | Preferred method for fetching object lists.          |
| Bucket    | GetBucketLocation       | GET    | /{bucket}       | location                | Returns the region (e.g., us-east-1).            |
| Bucket    | GetBucketVersioning     | GET    | /{bucket}       | versioning              | Get versioning state of bucket.                  |
| Bucket    | PutBucketVersioning     | PUT    | /{bucket}       | versioning              | Enable/suspend versioning on bucket.             |
| Bucket    | PutBucketAcl            | PUT    | /{bucket}       | acl                     | Sets canned ACLs or XML-defined policies.        |
| Bucket    | GetBucketAcl            | GET    | /{bucket}       | acl                     | Get bucket ACL configuration.                    |
| Object    | PutObject               | PUT    | /{bucket}/{key} | —                       | Standard upload (single PUT).                    |
| Object    | GetObject               | GET    | /{bucket}/{key} | —                       | Download object; handles Range headers.          |
| Object    | DeleteObject            | DELETE | /{bucket}/{key} | —                       | Remove single object.                            |
| Object    | DeleteObjects           | POST   | /{bucket}       | delete                  | Delete multiple objects (up to 1000) in one request. |
| Object    | HeadObject              | HEAD   | /{bucket}/{key} | —                       | Get metadata (size, content-type) only.          |
| Object    | CopyObject              | PUT    | /{bucket}/{key} | —                       | Triggered by x-amz-copy-source header.           |
| Object    | GetObjectAcl            | GET    | /{bucket}/{key} | acl                     | Get object-level ACL.                            |
| Object    | PutObjectAcl            | PUT    | /{bucket}/{key} | acl                     | Set object-level ACL.                            |
| Object    | ListObjectVersions      | GET    | /{bucket}       | versions                | List all versions of objects (versioned buckets).|
| Multipart | CreateMultipartUpload   | POST   | /{bucket}/{key} | uploads                 | Returns an UploadId for subsequent parts.        |
| Multipart | UploadPart              | PUT    | /{bucket}/{key} | partNumber=X&uploadId=Y | Upload specific chunk.                           |
| Multipart | UploadPartCopy          | PUT    | /{bucket}/{key} | partNumber=X&uploadId=Y | Copy data from another object as a part. Uses x-amz-copy-source header. |
| Multipart | CompleteMultipartUpload | POST   | /{bucket}/{key} | uploadId=Y              | Finalizes upload; triggers part merging.         |
| Multipart | AbortMultipartUpload    | DELETE | /{bucket}/{key} | uploadId=Y              | Cleanup of uncommitted parts.                    |
| Multipart | ListMultipartUploads    | GET    | /{bucket}       | uploads                 | List all incomplete multipart uploads in bucket. |
| Multipart | ListParts               | GET    | /{bucket}/{key} | uploadId=Y              | List parts uploaded so far for this upload ID.   |

### S3 API Operations Implementation Tier

**Tier 1 (Must Have):**
- All Service operations (ListBuckets)
- Core Bucket ops (Create, Delete, Head, ListObjectsV2, GetBucketLocation)
- Core Object ops (Put, Get, Delete, Head, CopyObject)
- All Multipart ops

**Tier 2 (Highly Recommended):**
- DeleteObjects (bulk delete efficiency)
- Versioning operations
- ACL operations

**Tier 3 (Nice to Have):**
- Additional bucket configuration operations
