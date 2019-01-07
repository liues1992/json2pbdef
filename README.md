# json2protodef
create protobuf definition from json data

# Usage

```
json2protodef '{"msg":"hello"}'

# output
message Message {
    string msg = 1;
}
```

```
json2protodef input.txt

json2protodef http://jsonapi.com/some_json_response

cat text.json | json2protodef

```
