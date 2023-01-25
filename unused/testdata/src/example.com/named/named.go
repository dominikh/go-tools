package pkg

type t1 struct{} //@ used(true)
type T2 t1       //@ used(true)
