# ugdvesting
The vesting module for the Unigrid network.

# Set vesting data

The endpoint to set vesting data is `https://127.0.0.1:39886/gridspork/vesting-storage/<address>`

The header requires `privateKey`

The body is set with the following data.

<amount> is the total amount being added to the vesting schedule
<start> the time the vesting begins
<duration> the length between vesting periods ISO 8601 duration format. For one month on average it's `P30DT10H` (30 days and 10 hours)
<parts> the total vesting periods

```bash
{
    "amount": "1000000",
    "start": "2023-08-29T16:53:46Z",
    "duration": "PT3H",
    "parts": 24
}
```