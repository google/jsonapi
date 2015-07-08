# jsonapi

A serailizer/deserializer for json payloads that comply to the
[jsonapi.org](http://jsonapi.org) spec in go.

## Background

You are working in your Go web application and you have a struct that is
similar to how your datbase table looks.  You need to send and receive
json payloads that adhere jsonapi spec.  Once you realized that your
json needed to take on this special form, you went down the path of
creating more structs to be able to serialize and deserialize jsonapi
payloads.  Then more models required these additional structure.  Ugh!
In comes jsonapi.  You can keep your model structs as is and use struct
field tags to indicate to jsonapi how you want your response built or
your request deserialzied.  What about my relationships?  jsonapi
supports relationships out of the box and will even side load them in
your response into an "included" array--that contains associated
objects.


