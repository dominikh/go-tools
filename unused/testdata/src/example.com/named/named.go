package pkg

type t1 struct{} //@ used("t1", true)
type T2 t1       //@ used("T2", true)
