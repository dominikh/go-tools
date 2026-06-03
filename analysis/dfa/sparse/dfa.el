(require 'cl-lib)

(defun format-state (prefix state ⊤ ⊥)
  (cond ((string= state "⊥") ⊥)
		((string= state "⊤") ⊤)
		(t (format "%s%s" prefix state))))

(defun dh/orgtbl-to-dfa-binary-table (table params)
  (let* ((table (--filter (not (equal 'hline it)) table))
		 (rows (1- (length table)))
		 (cols (1- (length (nth 0 table))))
		 (prefix (plist-get params :prefix))
		 (var (plist-get params :var))
		 (⊤ (plist-get params :⊤))
		 (⊥ (plist-get params :⊥)))

	(concat
	 (if var (concat "var " var " = ") "")
	 (format
	  "dfa.BinaryTable(%s, map[[2]%s]%s{\n"
	  ⊤ prefix prefix)
	 (mapconcat
	  (lambda (rowIdx)
		(mapconcat
		 (lambda (colIdx)
		   (let* ((x (nth 0 (nth rowIdx table)))
				  (y (nth colIdx (nth 0 table)))
				  (z (nth colIdx (nth rowIdx table))))
			 (format "{%s, %s}: %s," (format-state prefix x ⊤ ⊥) (format-state prefix y ⊤ ⊥) (format-state prefix z ⊤ ⊥))))
		 (number-sequence 1 cols)
		 "\n"))
	  (number-sequence 1 rows)
	  "\n\n")
	 "\n})")))
