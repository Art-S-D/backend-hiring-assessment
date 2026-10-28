# Report

## 1. Fix `GetAdByID`

The usecase was calling the DAO with `0` instead of the requested id. I passed the id through and removed the commented-out legacy code.

## 2. Delete ad endpoint

- DAO: `DeleteAdByID` returns `ErrNotFound` when no row is deleted.
- Usecase: inside a transaction, it loads the ad, checks that the caller owns it (`ErrForbidden` if not), then deletes it.
- HTTP: `DELETE /ads/{id}` (authenticated) returns `204`, `400` for a bad id, `403` when the caller isn't the owner, and `404` when the ad doesn't exist.

## 3. Username in `GetAdByID`

The usecase fetches the ad and then the owner, and returns both as `GetAdByIDResponse`.
If the owner no longer exists, the ad is still returned with an empty username and a warning is logged. Any other users-service error fails the request.

## 4. Rename `price_cents` to `price` (float)

I would push back on this change and didn't implement it, for two reasons:

- It is a breaking change: every existing consumer of the API (web, mobile, partners) reads `price_cents`.
- Floats can't represent every decimal amount exactly, which leads to rounding errors. Storing money as an integer number of the smallest currency unit is the usual practice; Stripe's API does this, for example.

First, I would ask the stakeholders why they need it. If it's only a display concern, clients can divide by 100 on their side.

If it's really needed, I wouldn't rename the field. I'd add a new `price` field next to `price_cents`.

## 5. Bulk import duplicates

Ads used to be created one by one, outside any transaction. A failure partway through left the first ads in the database, so a retry created them again.
I moved the import into its own `ImportAds` usecase, which creates all the ads in a single transaction. If any insert fails, nothing is written, so the client can retry safely.

This doesn't cover a retry after a successful import whose response was lost (timeout, network error): an idempotency key sent by the client would make that case safe too.

## 6. Publication quota (5 ads per rolling 24h)

- New `ad_publications` table (migration `003`) that logs every ad published through `POST /ads`, indexed on `(user_id, published_at)`. It has no foreign key to `ads`, so deleting an ad doesn't give the quota back.
- `CreateAd` runs in a transaction: it counts the user's publications in the window, rejects with `ErrQuotaExceeded` when the limit is reached, then creates the ad and records the publication.
- The HTTP layer maps that error to `429 Too Many Requests`.
- The limit and the window are configurable (`MAX_PUBLICATIONS`, `PUBLICATION_WINDOW`; defaults are 5 and 24h). The bulk import is not affected.

Limitations, accepted for a v1:

- Concurrent requests from the same user can both pass the check and exceed the quota by a few ads. Writes are rare, so this is unlikely.
- Nothing purges `ad_publications` yet; rows older than the window are useless and should be deleted periodically.
- At a larger scale, I would move the quota to Redis (e.g. a sorted set per user with a TTL), which fixes both points.

## Notes

- **Users client:** the provided `users/pkg/api.Client` doesn't implement the `users/pkg.UserClient` interface. Since `/users` must not be modified, I wrote a small adapter in `ads/internal/cmd/ads-api/usersclient`.
- **Transactions:** delete ad, bulk import and create ad (quota) each run in a transaction opened by the usecase. Moving that work into single DAO methods would save some round trips to the database, but I chose to keep the business logic in the usecases for now.
