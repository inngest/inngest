((go-mode . ((dape-configs .
                           ((go-debug
                             modes (go-mode go-ts-mode)
                             command "dlv"
                             ;; command-args ("dap" "--listen" "127.0.0.1:55878" "--log")
                             command-cwd default-directory
                             host "127.0.0.1"
                             port 40000
                             :type "go"
                             :mode "remote"
                             :request "attach"
                             :showLog "true")
                            (go-test
                             modes (go-mode go-ts-mode)
                             command "dlv"
                             command-args ("dap" "--listen" "127.0.0.1::autoport")
                             command-cwd default-directory
                             port :autoport
                             :type "go"
                             :mode "test"
                             :request "launch"
                             :showLog "true"
                             :program "."
                             :args [])
                            )))))
