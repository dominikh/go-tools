;;; go-keyify.el --- keyify integration for Emacs

;; Copyright 2016 Dominik Honnef. All rights reserved.
;; Use of this source code is governed by a BSD-style
;; license that can be found in the LICENSE file.

;; Author: Dominik Honnef
;; Version: 1.0.0
;; Keywords: languages go
;; URL: https://github.com/dominikh/go-keyify
;;
;; This file is not part of GNU Emacs.

;;; Code:

(require 'json)

;;;###autoload
(defun go-keyify ()
  "Turn an unkeyed struct literal into a keyed one.

Call with point on or in a struct literal."
  (interactive)
  (let ((res (json-read-from-string
              (shell-command-to-string (format "keyify -json %s:#%d"
                                               (shell-quote-argument (buffer-file-name))
                                               (1- (position-bytes (point)))))))
        (point (point)))
    (delete-region
     (1+ (cdr (assoc 'start res)))
     (1+ (cdr (assoc 'end res))))
    (insert (cdr (assoc 'replacement res)))
    (indent-region (1+ (cdr (assoc 'start res))) (point))
    (goto-char point)))

(provide 'go-keyify)

;;; go-keyify.el ends here
