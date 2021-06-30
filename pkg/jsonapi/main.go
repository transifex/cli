/*
Package jsonapi
Interface for interacting with {json:api} APIs.

Usage:

    import "github.com/transifex/cli/pkg/jsonapi"

    api := jsonapi.Connection{Host: "https://foo.com", Token: "XXX"}

    // Lets get a list of things
    query := jsonapi.Query{
		Filters: map[string]string{"age__gt": "15"},
	}.Encode()
    page, err := api.List("students", query)
    for {
        for _, student := range page.Data {
            fmt.Println(student.Attributes["full_name"])
        }
        if page.Next == "" {
            break
        } else {
            page = page.GetNext()
        }
    }

    // Lets get and manipulate a single thing
    teacher, err := api.Get("teachers", "1")
    teacher.Attributes["age"] = teacher.Attributes["age"] + 1
    err = teacher.Save([]string{"age"})

    // Lets fetch some relationships
    relationship, err := teacher.Fetch("manager")
    fmt.Println(relationship.DataSingular.Attributes["grade"])

    relationship, err = teacher.Fetch("students")
    page := relationship.DataPlural
    for {...}  // Same as before

    // Lets create something new
    student := jsonapi.Resource{
        API: api,
        Type: "students",
        Attributes: map[string]interface{}{
            "full_name": "John Doe",
        },
    }
    err = student.Save()  // Student has no ID so a POST request is sent

    TODOs:

    - Change/Reset/Add/Remove methods for relationships
*/
package jsonapi
