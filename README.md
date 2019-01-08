# json2pbdef
create protobuf definition from json data

# Usage

```
json2pbdef '{"msg":"hello"}'

# output
message Message {
    string msg = 1;
}
```

```
json2pbdef input.txt

json2pbdef http://jsonapi.com/some_json_response

cat text.json | json2pbdef

```

# TODO
- List rules that the json data must obey
- Add more tests
