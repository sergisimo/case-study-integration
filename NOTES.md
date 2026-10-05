# Review notes

## What to fix

1. Configuration should be loaded from environment variables instead of a static config file so that sensitive information like API keys are not stored in plaintext and can be managed securely by the deployment environment. (Since this requires big changes and external dependencies I have avoided fixing it.)
  1.1. Avoid logging sensitive configuration details such as API keys, there is some libraries that allow tagging certain fields as sensitive so that they are redacted in logs.
  1.2. `config.yaml` says that the secrets are encrypted but they are base64-encoded, which is not encryption. If you want to keep the secrets secure you could use `sops` paired with a cloud key vault to encrypt the secrets so they can live safely in the repository. Better if the secrets are also moved to the key vault.
  1.3. `config.yaml` assumes the values are encrypted but then when reading from the file they are not decrypted, this means that when they are used in the endpoints the API Keys will be incorrect.
2. In a lot of different places throughout the code the errors are ignored using _. Errors should never be ignored and they always need to be handled, the function that finds them can either do something to recover from the error, log it, or propagate it up the call stack via return for proper handling. (This has been fixed everywhere).
  2.1. `Fatalf` should only be used once the errors have been handled properly otherwise we break the normal flow of the program, if on record fails to be stored it should be recorded/logged somewhere and we should try to decode the other records instead of terminating the program immediately. (This has been fixed).
3. `io.ReadAll` should only be used in very specific cases since it reads the entire response body into memory without streaming, on the example where it is used for the error message it's not a big deal. But when reading the full response body and unmarshalling we should use the `json.Decoder` to stream the data and handle errors properly. All the data will still be stored in memory but in this case it will only be once, with the ReadAll we would use approximately twice the memory. (This has been fixed).
  3.1. Similar to ReadAll, when writing the json into the log files since `*os.File` is already a Writes, using `json.Encoder` would be more efficient and handle errors properly. In this case we haven't changed it since I have prioritized the redaction of sensitive information.
4. Do not use global variables for tracking state such as `recordsProcessed` and `recordsFailed`. This variable can be modified by any piece of code in the same package, leading to potential unwanted side effects. (This has been fixed)
5. When logging records for debugging purposes, ensure that sensitive information is redacted. The `MarshalRedactedJSON` method should be used to avoid any data that could expose personal or sensitive information. (This has been implemented)
6. When sending the records to the Recare API, since it involves Network I/O blocking operations the best here is to use go routines to optimize the sending and avoid blocking the main execution flow. Since this is an external call that could be temporarly offline it is important to also implement a retry logic with exponential backoff to handle transient failures gracefully. (This has been implemented)
  6.1. The retry logic should check for permanent errors but since we do not have a clear API specification here for now I count all errors as potentially transient and retry accordingly.
  6.2. Although Goroutines are very lightweight, the concurrency implemented is unbounded. In case the number of records was very large it would be better to switch into a fixed-sized worker pool. 
7. The HTTP clients when doing requests to external services did not have a timeout set. It's important to set a timeout to avoid hanging requests and potential resource leaks. This can be done by using `context.WithTimeout` when creating the request or by configuring the `http.Client` with a timeout. (This has been fixed)
  7.1. `context.Context` was not used in the original implementation, idiomatic Go encourages passing the context down to all functions that perform I/O or other operations that may need to be canceled or have timeouts. This allows for better control over request lifecycles and resource management.

## Transform Fixes

The transform function had two main issues and an optional improvement:
- First the condition where not split by commas as the data definition expected. This lead to having the comman separated values as the first element in the array, causing incorrect parsing of the conditions.
- Second, the transform function did not properly handle when the weight was provided in pounds instead of kilograms. The function needs to correctly convert the weight to kilograms to ensure accurate data representation.
- Third, it's always better to use time.Time to work with dates instead of strings, as it allows for proper date manipulation, comparison, and formatting.

## AI Usage

- I have researched, written and fixed all the issues manually with the help of the AI autocomplete because I wanted to test myself to see if I could find as many issues and improvements as possible.
- I have used AI Agent to generate the `record_test.go` which I reviewed and didn't have to modify anything since it does test what I intended it to test.
- Finally I used the AI to review and suggest improvements, it found some small improvements like I forgot one error handling or the suggestiong 6.1 and 6.2.


## Code Restructuring

For the code restructuring I focused on changing the structure to have a clear separation of layers: 
- This way we have a transport layer for the HIS communication with the legacy hospital system, this layer allows to isolate the specific protocol and implementation details of the HIS so it does not spill over the rest of the system. 
  - This change also alows for easier testing and mocking both of the package and the rest of the system. This is because we can test the package in isolation using the public API allowing us to test the transform function in isolation. The rest of the system now is also easier to test since we can mock the transport layer and focus on the business logic in the domain layer.
- I've also created the `discharge` package that works as our domain layer in this small application, it defines the struct we want to be working on the domain level and encapsulates the business logic related to discharge summaries. 
  - Since there is not of business logic in this package I've kept the client simple and focused on the transport layer, but in case we needed more complex business logic in the future, I would split up the domain, transport and repository layers inside the same package. I like to work inside the same package and same files even if we are in different layers because Go idiomatically encourages keeping the files and packages split by domain and not by layers.
