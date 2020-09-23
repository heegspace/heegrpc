namespace go example

struct example_req {
    1:i32       reqcode
    2:string    reqvalue
}

struct example_res {
    1:i32       rescode
    2:string    resvalue
}

service example_service {
  example_res get_response(1:example_req req)
}