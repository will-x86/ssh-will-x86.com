Initially ran:
```bash
npx wrangler kv namespace create receipt-paper
```


Output like:
```
 ⛅️ wrangler 4.77.0
───────────────────
✔Select an account › ... Account
Resource location: remote

🌀 Creating namespace with title "receipt-paper"
✨ Success!
To access your new KV Namespace in your Worker, add the following snippet to your configuration file:
{
  "kv_namespaces": [
    {
      "binding": "receipt_paper",
      "id": "IDHERE"
    }
  ]
}

```



Added a secret:


```bash
openssl rand -hex 32
```

e.g.:
```
2839ed77e6a4dcc9861af907ffddd2745f44f75a16c06b693ac8bc70e5b7e529
```



Then added the secret:

```
npx wrangler secret put WORKER_SECRET

 ⛅️ wrangler 3.114.17 (update available 4.77.0)
---------------------------------------------------------
// Boring update message

✔ Select an account › .... Account
✔ Enter a secret value: … ************ etc
🌀 Creating the secret for the Worker "receipt-paper"
✔ There doesn't seem to be a Worker called "receipt-paper". Do you want to create a new Worker with that name and add secrets to it? … yes
🌀 Creating new Worker "receipt-paper"...
✨ Success! Uploaded secret WO

```



Needed to deploy the worker:
```
npx wrangler deploy

 ⛅️ wrangler 3.114.17 (update available 4.77.0)
// Boring update message

Total Upload: 2.33 KiB / gzip: 0.86 KiB
Your worker has access to the following bindings:
- KV Namespaces:
  - MESSAGES: XXXXX
Uploaded receipt-paper (7.52 sec)
Deployed receipt-paper triggers (3.19 sec)
  https://receipt-paper.XXXXX.workers.dev
Current Version ID: XXX
```
