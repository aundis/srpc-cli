package packed

import "github.com/gogf/gf/v2/os/gres"

func init() {
	if err := gres.Add("H4sIAAAAAAAC/5SXeTjU+/vGP3YxNNmXI0NkC2PSZPvSiMKEUY0WlXXKFhpDdhWphkGWqXSmKeFEyHZaHZO1IkZ2yq4xwljK2uB36Vwdyxk6v/ljPjPXzPW6nut5nvd9v28UkoNTBOAFAOD27su2wKoXGNgC+GC93THOOE03LxwG6+XoiT7KBbBBqEouKCQP7+o/r2CerMOIscBoenqfc3P+AbPYFDa/Dia9IUzTF+vzN7F9EyIvwCd+ZQ1RaXPijzeNc94auADcD3oTQ9kl27bW69NTYaMx9u+CX8XB+2NEVWdlCaqmeXChZzcE2tyCedkjiAdgwjeOmqleyGtzjjP1Do2ZZsMbWn2YVsfLvXjVkTMqhQgdx/eFMGl77at9lRcLKe3aVMdctUhuUanevCs11j5e80I0zdhSt06sjsHVxMZjPrgAJj6YEL47eInSjoOPms5ZafO2fFMkYVo9+iEkS6V3Th+G3F0nbgrJaNx6cVSQIHTe60NIvOLOi/x3St7rPzCJqEIST+oW+y9KojPVMlOYOc2qxVoHpIdlhzV4VXBzyb6hsLvsCi2SmSmQXpeDXUP5NxnuzwfSGVitRxYdxfuVEu4+4W1koB/e1fP+wixf+tpxXfArTXDLlaakHJPPjPgGuLfFRdL7jJh9LRGqCNWhw9zR7Tin3vvAVVmVRpWH4petw6IIRYIESh6u0TXvE0hVMZjJaZQRU5uBLDjpjpFprqpkfhsYcF380kcfAfGZKZvxSiDceNJt+ZCDc1hyEOcw8r3e7qQ6DnzZVvNE6oKc3kR9K6rXEaUoE+EF5Ztm3q3E7iJ3KL+9dP8WQ8qJMWfA5N7+yAkGRmNg9IqpU1urMZeqwEbcpntas4gCkhANl9GxxWOXEnL2QoeuJmDwVfULCfJy0mmLAXBSUwdnvkQ3gk3rYsXAxYDA3JqL9BC+nRWHucU9lmzGUXi44vXvfoOzgd5pWL+UEPO2aB7/J+mRKnYGeebccvePP2KqqaOlpC8Z7Z8pi753XfB8NcysIDIf9Oj1nK+IaEplw44XcnXg1sWScOJUirpQECg4GNwXVvCC1z0xilJvWT27ze+Y5aGz+ftAVS8lhjO2qhI9E0qv3Hw9acuc2bvNc9fCkrq2M1RN34pjil1qvL8HMjHEu7MkwOQ0ldwiaPlFlvouvFIeUW2+ByJDlCNzqoiUVUweZ78qi0taRNHczCTnzjkjwmJj3ZvpT0131/XRcizbn9Y1ZkkNKVDzRz/4ZMUrcNQ8YQbuyO+v39Jvm17SbPWRNE61ndSqyOIfLTLMrkHk6tV8lerQpPZS7cMWFt6QF+cXrxmFfdeFDzxLlfHp5BIgtybln6MfmOtTuP2pIRGncUV2WMnWmC2gKzSiosmY/fs3iWg5lGSUxQgGP3DG1Qycs/DbPpuprtu2kV38x5Ryqbbqk7o9VfMzB+gqGg4kizCGR+lWXXtpruwbMt05yPRLOZC5fD5h7OfPVbrhUWM8RYwM9trXPd/r/W2+oHRKIaELtMlEfoVci8MteW7N25AX2pXbzzrn+hO73beYDPl5EF4HQkeIkFi9QCRnKu7wY8u4O2fpKF3+332XSo9EC4kjxV4TZh4mjKp3+b8prb2n1svJmSGRtsPerjgUXFXIhEqkW/pmz5fWZiQeT/1GqqTYkKDTFkUqd2odNOMvHDnNgG6/Tbhn8nFe9QS1orHm2fGXt+gmtwPhGA9HqfDxeYHQpAnCqU914pJm10jdM8Z76JaafUKg/GKR3OIUD7S5BcKiz/AG0yMza9xELbmSMNgpCpnfjk5uUHnu8LgSXAalG5XpfaAImofuTm2xs2B+MhyfLRwtg7kWdlZ08Yn5y1N7SDyGUh0gjT+9jgnUp1qYR5Ym5n18h48jCpnstZPMMvXQ6ovuqP6q1bIzKiRoaAY+Olst2ZFa0/5tDACApaVl+TxKb3br5QSAUdBq+dTZulY+lX8hn7hAH4zvav38rVfZJTueYlUOBV8bPDmp/7seTFtf6AiHfKo2Rzv48JPpbZ+MnmLso3inOm1y0tyRYMn5j3WLDEZhIWi7l5hKAYhmJdzkogJJjPLj9y5Vu63J5zDov/st8jhR1IBmugcps0SQkVVJG3/09LhwRU9zPq1FOGlYQCUsg/THOP4QWFcb32uHfxedL3VYnsfZ43m1uzDpL7IXobkxWSRRkeHG6X3B+ka4e7ZTv2fkTNyrEp0jQ7PU7fpwGSsah8A5htItq89BF5uDLMZeFQ8ydLR0YYLv698GZE4mi4jtxDqGllMMFn0k2rqXPiPw5H9aeV0LfnQEAIDL7Ku9jU17rbdJsGqlLwbr7+aM+dG6vk2NLWOdVSpsgvuXq+lQlVyy40vxH6HgKKoYXSYspKD0rTU4yOypIuplXqfdEOLmm9xH6oFv+jH7374SV8moTS+6RoqeGLvjXXLho9OMVzr6o5Osftm+y218fjqkCZUwF/1t0+S4eImDEeU7SgwM+3t9QuCBdPuBJc2copSc7248ZcbMLzLpb619XRDpaJuXEpFdaYcCbQTPhGUr9fc1dezBhbmJ3zudiSaV1ATu/I0yOdITAZ5p2aZxUtT6W0GDfHLiXs99hZT/SUP/gti8b/HLc/I42l1CrjMWuK6LzUFPQOco4GxhWQ4xR9kobuI1v1QIf4wWxVSjdb6q4UuKi726c0xdbPteuz+e20fsD8r8MpqEuzQ2FnRRn9OIVnz7/Hiu6oJ2XJK/vAlCtrX+QVzIjDTssXkvmOtQhH3k7P47Gq/c0dB31b1UYi1s2OB854mabpRfV7gSF/lgGNfPuVPE+DFWbACQtmbuIuvmLspyUD9vM7z1m92P1rOkNmJpYs674f4DkHsdUGZD4HlHL8dzGOwP5simm2mwTjN2/Yr587l6QxWWdSOB6t4Plaigas3KGGIL5Nqse8oTdt3dBzf09JK7ptnwLNdueGvBKeUHY2dc/oxN5WzY79fkGlYec5qIWmgzTyyITO/jFDCHud4oHxJYlKx1eNF5CZ2l5oAuz+0qH2s+ax8Te7BtjKxtLkwY4ZJmvywkDWNkiZ6U5D+R8p5m6UGOOlKapHwhvngUiRAbnEnvcoVbJ35CnPmwr3WaOCDINeWjM91SVDCB1/ufcGH4u74UPcuqIcFTTYk9sjDFHJvTg/eOQlCyBjR6Af5zWq7NZGzdmMb8EkSS2cdMPfvW87FCwJj8/bLTVoKD2ne42nm26hiLBjQOqHrp/1mrwDcP+vrI5wYfyiPBgF9DvGdXoEn2w2TYZ/gVV2Wyzyt6ku8OdQfO05ULcsEnHlAnr9nFZUaNOLjMJsvTMCJvmPNbjL3zjAcIw8HGTYYmnSIyl2XZfi7pzUNTEjZsAFDEsWax1l2TeVZm9mMcdIYyi4mzsYtwbBwmfn5aQiw/N4kWKxhWYWIFc5AF5l/RYgXGKkyswJI2hK2PFj+Jf4eJ1T6p9A8RAJ5epm1OZBktVtOXvXa1dSivoStyA/9/513dDlb+s9IOHx7gv7jRSrXL/rNa8BTWVEvZBMfSjVZXykoxVyp13QL8Uj83ZkmtYRVtxFqvnytAVoq5AuTiA/6zfq70clkxV5/LXWt6eeJXzI30c00fWBzwlbIFQACL487Fvfw7CAAB3RwAgAQtf/u/AAAA//+Ero22QBAAAA=="); err != nil {
		panic("add binary content to resource manager failed: " + err.Error())
	}
}
