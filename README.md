# Fauna Store

A [Gorilla sessions](https://github.com/gorilla/sessions) store implemention backed by FaunaDB.

## Example

```
client := f.NewFaunaClient(os.Getenv("FAUNA_PASS"))
store, err := NewFaunaStore(client)
```

## Assumptions

This library assumes the existence of a `sessions` collection in the database to which it connects.

The tests assume you have an environment variable `FAUNA_PASS` defined with the connection info for FaunaDB.  
If running the tests, be aware that they don't cleanup generated database records.