Adapted from [cronexpr](https://github.com/gitploy-io/cronexpr)

# Cron

A cron expression parser and next occurrence calculator

## Usage

```golang
import "time"
import "github.com/amasanelli/cron"

next := cron.MustParse("0 0 1 * 1", time.UTC).Next(time.Now())
```

### Parse(cronExpression, timezone)
Parses the cron expression and sets it's timezone.
The cron expression must be written for the desired timezone; e.g., if the cron should run every day at 00:00 in Melbourne the cron expression should be `"0 0 * * *"` and the 
```golang 
timezone, _ = time.LoadLocation("Australia/Melbourne")
```
It throws an error in case of failure.

### MustParse(cronExpression, timezone)
Does the same as Parse, but it panics in case of failure

### Next(referenceTime)
Calculares the next occurence for the cron expression and the given time. It converts the input to the timezone setted in the Parse/MustParse function to perform the calulation

## Implementation

```
Field name     Mandatory?   Allowed values    Allowed special characters
----------     ----------   --------------    --------------------------
Minutes        Yes          0-59              * / , -
Hours          Yes          0-23              * / , -
Day of month   Yes          1-31              * / , - 
Month          Yes          1-12              * / , -
Day of week    Yes          0-6               * / , - 
```

### Asterisk (`*`)
Asterisks indicate that the field matches all the allowed values; e.g., using an asterisk in the 4th field (months) means every month.

### Slash (`/`)
Slashes are used to indicate steps; e.g., */15 in the 1st field (minutes) means that the cron will run every 15 minutes

### Comma (`,`)
Commas are used to separate items of a list; e.g., 5-6,0-1 in the 5th field (dow) could be used to indicate a cron that runs from Friday to Monday

### Hyphen (`-`)
Hyphens define ranges of values; e.g., 0-6 in the 2nd field (hours) indicates the range of hours from 0 to 6, inclusive
