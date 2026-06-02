// 表单提交处理
document.addEventListener('DOMContentLoaded', function() {
    // 登录表单提交
    const loginForm = document.getElementById('loginForm');
    if (loginForm) {
        loginForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            
            // 获取表单数据
            const username = document.querySelector('input[name="username"]').value.trim();
            const password = document.querySelector('input[name="password"]').value.trim();
            
            // 验证
            if (!username || !password) {
                alert('用户名和密码不能为空！');
                return;
            }
            
            const formData = new FormData(loginForm);
            const params = new URLSearchParams(formData);
            
            try {
                const response = await fetch('/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: params.toString()
                });
                
                if (response.ok) {
                    window.location.href = '/welcome';
                } else {
                    alert('登录失败，请检查用户名和密码');
                    window.location.href = '/login';
                }
            } catch (error) {
                console.error('登录失败:', error);
                alert('登录出错，请稍后重试');
            }
        });
    }
    
    // 注册表单提交
    const regForm = document.getElementById('registerForm') || document.getElementById('regForm');
    if (regForm) {
        regForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            
            // 获取表单数据
            const username = document.querySelector('input[name="username"]').value.trim();
            const password = document.querySelector('input[name="password"]').value.trim();
            const password2 = document.querySelector('input[name="password2"]') ? 
                             document.querySelector('input[name="password2"]').value.trim() : password;
            
            // 验证
            if (!username || !password) {
                alert('用户名和密码不能为空！');
                return;
            }
            
            if (password2 && password !== password2) {
                alert('两次输入的密码不一致！');
                return;
            }
            
            if (password.length < 6) {
                alert('密码长度不能少于6位！');
                return;
            }
            
            const formData = new FormData(regForm);
            const params = new URLSearchParams(formData);
            
            try {
                const response = await fetch('/register', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: params.toString()
                });
                
                if (response.ok) {
                    alert('注册成功！请登录');
                    window.location.href = '/login';
                } else {
                    alert('注册失败，用户名可能已存在');
                    window.location.href = '/register';
                }
            } catch (error) {
                console.error('注册失败:', error);
                alert('注册出错，请稍后重试');
            }
        });
    }
    
    // 加载图片库
    if (document.getElementById('gallery')) {
        loadGallery();
    }
    
    // 加载视频库
    if (document.getElementById('videoGallery')) {
        loadVideos();
    }

    // 注册页交互：密码强度与启用提交按钮
    const passwordInput = document.getElementById('password');
    const confirmInput = document.getElementById('confirmPassword');
    const usernameInput = document.getElementById('username');
    const strengthMeter = document.getElementById('strengthMeter');
    const strengthText = document.getElementById('strengthText');
    const registerBtn = document.getElementById('registerBtn');

    function calcPasswordScore(pw) {
        let score = 0;
        if (!pw) return 0;
        // length
        score += Math.min(10, pw.length) * 2;
        // variety
        if (/[a-z]/.test(pw)) score += 10;
        if (/[A-Z]/.test(pw)) score += 10;
        if (/[0-9]/.test(pw)) score += 10;
        if (/[^A-Za-z0-9]/.test(pw)) score += 15;
        return Math.min(100, score);
    }

    function updateStrength(pw) {
        if (!strengthMeter || !strengthText) return;
        const score = calcPasswordScore(pw);
        strengthMeter.style.width = score + "%";
        strengthMeter.style.transition = 'width 180ms ease';
        if (score < 30) {
            strengthMeter.style.background = 'var(--accent-red)';
            strengthText.textContent = '密码强度：弱';
        } else if (score < 60) {
            strengthMeter.style.background = '#ffb142';
            strengthText.textContent = '密码强度：中等';
        } else {
            strengthMeter.style.background = '#2ed573';
            strengthText.textContent = '密码强度：强';
        }
    }

    function validateUsername(name) {
        return /^[A-Za-z0-9_]{3,20}$/.test(name);
    }

    function refreshRegisterState() {
        if (!registerBtn) return;
        const usernameValid = usernameInput ? validateUsername(usernameInput.value.trim()) : false;
        const pw = passwordInput ? passwordInput.value : '';
        const pwOk = pw && pw.length >= 6;
        const match = confirmInput ? (confirmInput.value === pw) : true;
        if (usernameValid && pwOk && match) registerBtn.removeAttribute('disabled');
        else registerBtn.setAttribute('disabled', '');
    }

    if (passwordInput) {
        passwordInput.addEventListener('input', function(e){
            const pw = e.target.value || '';
            updateStrength(pw);
            const passwordError = document.getElementById('passwordError');
            if (passwordError) passwordError.style.display = (pw.length >= 6) ? 'none' : 'block';
            refreshRegisterState();
        });
    }

    if (confirmInput) {
        confirmInput.addEventListener('input', function(e){
            const pw = passwordInput ? passwordInput.value : '';
            const okEl = document.getElementById('confirmPasswordSuccess');
            const errEl = document.getElementById('confirmPasswordError');
            if (e.target.value === pw) {
                if (okEl) okEl.style.display = 'block';
                if (errEl) errEl.style.display = 'none';
            } else {
                if (okEl) okEl.style.display = 'none';
                if (errEl) errEl.style.display = 'block';
            }
            refreshRegisterState();
        });
    }

    if (usernameInput) {
        usernameInput.addEventListener('input', function(e){
            const okEl = document.getElementById('usernameSuccess');
            const errEl = document.getElementById('usernameError');
            if (validateUsername(e.target.value.trim())) {
                if (okEl) okEl.style.display = 'block';
                if (errEl) errEl.style.display = 'none';
            } else {
                if (okEl) okEl.style.display = 'none';
                if (errEl) errEl.style.display = 'block';
            }
            refreshRegisterState();
        });
    }
});

// 动态加载图片库
async function loadGallery() {
    try {
        const gallery = document.getElementById('gallery');
        const noImages = document.getElementById('noImages');
        
        // 预定义的图片路径列表
        const baseUrls = [
            '/image/pic1.jpg', '/image/pic2.jpg', '/image/pic3.jpg',
            '/image/pic4.jpg', '/image/demo1.jpg', '/image/demo2.jpg'
        ];
        
        let foundImages = 0;
        
        // 检测并加载存在的图片
        for (const url of baseUrls) {
            try {
                const response = await fetch(url, {method: 'HEAD', cache: 'no-cache'});
                if (response.ok) {
                    foundImages++;
                    const name = url.split('/').pop();
                    gallery.innerHTML += `
                        <div style="background:var(--bg-secondary);border:1px solid var(--border);border-radius:8px;overflow:hidden;transition:all 0.3s ease" 
                             onmouseover="this.style.borderColor='var(--accent-red)'; this.style.boxShadow='0 5px 15px rgba(230,57,70,0.2)'" 
                             onmouseout="this.style.borderColor='var(--border)'; this.style.boxShadow='none'">
                            <img src="${url}" style="width:100%;height:200px;object-fit:cover;display:block;cursor:pointer" alt="${name}" onclick="window.open('${url}', '_blank')">
                            <div style="padding:0.75rem">
                                <p style="margin:0;color:var(--text-secondary);font-size:0.9rem;text-align:center;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${name}</p>
                            </div>
                        </div>
                    `;
                }
            } catch (e) {
                // 图片不存在或加载失败，跳过
            }
        }
        
        if (foundImages === 0) {
            gallery.style.display = 'none';
            noImages.style.display = 'block';
        } else {
            gallery.style.display = 'grid';
            noImages.style.display = 'none';
        }
    } catch (error) {
        console.error('加载图片库失败:', error);
    }
}

// 动态加载视频库
async function loadVideos() {
    try {
        const videoGallery = document.getElementById('videoGallery');
        const noVideos = document.getElementById('noVideos');
        
        // 预定义的视频路径列表
        const baseUrls = [
            '/video/xxx.mp4', '/video/demo1.mp4', '/video/demo2.mp4',
            '/video/sample.mp4', '/video/test.mp4', '/video/movie.mp4'
        ];
        
        let foundVideos = 0;
        
        // 检测并加载存在的视频
        for (const url of baseUrls) {
            try {
                const response = await fetch(url, {method: 'HEAD', cache: 'no-cache'});
                if (response.ok) {
                    foundVideos++;
                    const name = url.split('/').pop();
                    videoGallery.innerHTML += `
                        <div style="background:var(--bg-secondary);border:1px solid var(--border);border-radius:8px;overflow:hidden;transition:all 0.3s ease" 
                             onmouseover="this.style.borderColor='var(--accent-red)'; this.style.boxShadow='0 5px 15px rgba(230,57,70,0.2)'" 
                             onmouseout="this.style.borderColor='var(--border)'; this.style.boxShadow='none'">
                            <video style="width:100%;height:200px;background:#000;object-fit:cover;display:block" controls preload="metadata">
                                <source src="${url}" type="video/mp4">
                            </video>
                            <div style="padding:0.75rem">
                                <p style="margin:0;color:var(--text-secondary);font-size:0.9rem;text-align:center;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${name}</p>
                            </div>
                        </div>
                    `;
                }
            } catch (e) {
                // 视频不存在或加载失败，跳过
            }
        }
        
        if (foundVideos === 0) {
            videoGallery.style.display = 'none';
            noVideos.style.display = 'block';
        } else {
            videoGallery.style.display = 'grid';
            noVideos.style.display = 'none';
        }
    } catch (error) {
        console.error('加载视频库失败:', error);
    }
}
