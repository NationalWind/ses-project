# Dự Án SES - Kịch Bản Video Demo

## Tổng Quan
Kịch bản này cung cấp hướng dẫn để quay video ~5-10 phút trình bày triển khai thuật toán SES (Hệ Thống Thực Thi Tuần Tự).

**Đối Tượng Mục Tiêu**: Giảng viên và sinh viên trong các khóa học hệ thống phân tán

---

## Danh Sách Kiểm Tra Trước Quay

- [ ] Xây dựng dự án: `go build -o ses.exe cmd/main.go`
- [ ] Xác minh không có quy trình nào đang chạy: `pkill -9 ses.exe`
- [ ] Xóa nhật ký: `rm -rf logs/`
- [ ] Chạy thử hoàn tất thành công
- [ ] Cửa sổ terminal ít nhất 100x40 ký tự
- [ ] Kích thước font dễ đọc (16pt+)
- [ ] Microphone hoạt động và đã kiểm tra
- [ ] Phần mềm ghi hình sẵn sàng (OBS, ScreenFlow, v.v.)
- [ ] Ghi hình màn hình độ phân giải 1080p hoặc cao hơn

---

## Phần 1: Giới Thiệu (1 phút)

**Lời Thoại:**
> "Xin chào, đây là một bản demo về thuật toán SES - Hệ Thống Thực Thi Tuần Tự - để sắp xếp tin nhắn phân tán. Dự án này triển khai một hệ thống phân tán gồm 15 quy trình trao đổi tin nhắn đồng thời duy trì thứ tự nhân quả bằng cách sử dụng vector clocks.
>
> Vấn đề chúng ta đang giải quyết: Trong các hệ thống phân tán, các tin nhắn từ nhiều người gửi có thể đến không theo thứ tự tại máy nhận. Chỉ xử lý chúng khi chúng đến có thể vi phạm nhân quả - chúng tôi có thể xử lý một tin nhắn trước khi xử lý các tin nhắn khác mà nó phụ thuộc vào.
>
> Thuật toán SES cung cấp một giải pháp thanh lịch bằng cách sử dụng thông tin vector clock được gắn thêm vào, đảm bảo các tin nhắn được gửi theo thứ tự nhân quả một cách nhất quán đồng thời giảm thiểu chi phí mạng."

**Trên Màn Hình:**
- Hiển thị file README.md
- Làm nổi bật phần "Vấn Đề Đang Giải Quyết"
- Hiển thị sơ đồ kiến trúc hệ thống

---

## Phần 2: Cấu Hình Hệ Thống (1-2 phút)

**Lời Thoại:**
> "Hãy xem xét cấu hình hệ thống. Chúng ta có 15 quy trình chạy trên localhost, sử dụng các cổng từ 8000 đến 8014. Mỗi quy trình sẽ gửi 150 tin nhắn cho mỗi quy trình khác trong 14 quy trình, với tổng cộng hơn 31.000 tin nhắn.
>
> File cấu hình chỉ định ID quy trình, địa chỉ mạng và cổng. Chúng tôi sử dụng TCP để gửi tin nhắn một cách đáng tin cậy."

**Trên Màn Hình:**
```bash
# Hiển thị cấu hình
cat config/config.json
```

**Các Điểm Cần Làm Nổi Bật:**
- 15 quy trình với ID duy nhất
- Localhost với các cổng khác nhau
- 150 tin nhắn cho mỗi điểm đến
- Tốc độ gửi 100 tin nhắn/phút

---

## Phần 3: Xây Dựng & Khởi Động (1 phút)

**Lời Thoại:**
> "Trước tiên, chúng ta sẽ xây dựng dự án. Hệ thống được viết bằng Go, cung cấp cho chúng ta quản lý tương tranh hiệu quả với goroutines."

**Trên Màn Hình:**
```bash
go build -o ses.exe cmd/main.go
```

Chờ cho quá trình xây dựng hoàn tất.

**Lời Thoại:**
> "Bây giờ chúng tôi sẽ khởi động tất cả 15 quy trình đồng thời. Mỗi quy trình sẽ tự động gửi tin nhắn đến tất cả các quy trình khác."

**Trên Màn Hình:**
```bash
bash send_all.sh
```

Hiển thị kết quả ban đầu:
```
Khởi động tất cả 15 quy trình với chế độ tự động gửi...
Khởi động quy trình 0...
Khởi động quy trình 1...
...
```

---

## Phần 4: Trao Đổi Tin Nhắn (2-3 phút)

**Lời Thoại:**
> "Hãy xem các tin nhắn được gửi và nhận. Hãy chú ý đầu ra bảng điều khiển hiển thị nhiều loại sự kiện:
>
> Đầu tiên, chúng ta thấy các tin nhắn 'SENT' - đây là các tin nhắn được truyền từ quy trình này sang quy trình khác với dấu thời gian vector của chúng.
>
> Sau đó, chúng ta thấy các tin nhắn 'RECEIVED' - chúng đến tại đích với trạng thái vector clock hiện tại của người gửi.
>
> Điều quan trọng nhất, chúng ta thấy các tin nhắn 'DELIVERED' - đây là các tin nhắn đã được gửi đến ứng dụng. Một tin nhắn chỉ được gửi khi tất cả các phụ thuộc nhân quả của nó đã được thỏa mãn.
>
> Quan trọng, chúng ta cũng thấy một số tin nhắn 'BUFFERED'. Đây là những tin nhắn đã đến trước khi các phụ thuộc của chúng được thỏa mãn, vì vậy chúng đang được giữ trong bộ đệm."

**Trên Màn Hình:**
Mở terminal khác và hiển thị nhật ký trực tiếp:
```bash
tail -f logs/process_0.log | grep -E "SENT|RECEIVED|DELIVERED|BUFFERED"
```

**Lưu Ý:**
- SENT: Tiêu đề tin nhắn với dấu thời gian như "P0-P1-M5"
- RECEIVED: Tin nhắn đến với thông tin vector
- DELIVERED: Tin nhắn được xử lý theo thứ tự
- BUFFERED: Tin nhắn chờ, với lý do như "thiếu phụ thuộc từ P2"

**Lời Thoại:**
> "Hãy xem ví dụ này: Quy trình 0 đã lưu vào bộ đệm một tin nhắn từ Quy trình 2 vì chúng ta chưa nhìn thấy tất cả các tin nhắn mà Quy trình 2 phụ thuộc vào. Khi những phụ thuộc đó đến, tin nhắn này sẽ tự động được gửi từ bộ đệm."

**Trên Màn Hình:**
```bash
grep "BUFFERED" logs/process_0.log | head -3
grep "DELIVERING FROM BUFFER" logs/process_0.log | head -3
```

---

## Phần 5: Sự Phát Triển của Vector Clock (2 phút)

**Lời Thoại:**
> "Hãy kiểm tra các vector clocks. Mỗi quy trình duy trì một vector clock - một bộ đếm cho mỗi quy trình trong hệ thống. Ban đầu, tất cả đều bằng không.
>
> Khi hệ thống chạy, những đồng hồ này theo dõi nhân quả. Ví dụ, khi Quy trình 0 gửi một tin nhắn, đồng hồ của nó tăng lên. Khi các quy trình khác nhận tin nhắn từ Quy trình 0, chúng cập nhật kiến thức về tiến độ của Quy trình 0."

**Trên Màn Hình:**
```bash
# Hiển thị một tin nhắn với thông tin vector clock đầy đủ
grep "SENT" logs/process_0.log | head -1
```

Làm Nổi Bật:
- ID Tin Nhắn: P0-P1-M1
- Dấu Thời Gian (tm): [1 0 0 ...] - Vector clock của Quy trình 0 tại thời điểm gửi
- Mục Vector_P (V_M): [(P1, [1 0 0...]), ...] - thông tin được gắn thêm

**Lời Thoại:**
> "Bạn thấy trường 'V_M' không? Nó là viết tắt của 'Vector Message' - nó chứa thông tin 'gắn thêm' về các tin nhắn khác mà chúng ta đã gửi. Đây là đổi mới chính của thuật toán SES.
>
> Thay vì gửi toàn bộ vector clock 15 thành phần với mỗi tin nhắn, chúng ta chỉ gửi thông tin phụ thuộc cần thiết. Điều này làm giảm chi phí mạng một cách đáng kể."

**Trên Màn Hình:**
Hiển thị sự phát triển vector clock:
```bash
echo "Trạng thái cuối cùng của Quy trình 0:"
tail -5 logs/process_0.log | head -3
```

---

## Phần 6: Phép Lưu Trữ Đệm (1-2 phút)

**Lời Thoại:**
> "Bây giờ hãy xem xét một kịch bản lưu trữ đệm cụ thể để hiểu cách thuật toán đảm bảo nhân quả.
>
> Hãy tưởng tượng Quy trình 1 nhận một tin nhắn từ Quy trình 2, và Quy trình 2 chỉ ra một phụ thuộc vào Quy trình 3. Nếu Quy trình 1 chưa nhìn thấy các tin nhắn bắt buộc từ Quy trình 3, tin nhắn này sẽ được lưu vào bộ đệm."

**Trên Màn Hình:**
```bash
# Tìm một ví dụ lưu trữ đệm
echo "=== Tin Nhắn Được Lưu Trữ Đệm trong Quy Trình 0 ==="
grep "BUFFERED" logs/process_0.log | head -5

echo ""
echo "=== Khi chúng được gửi từ bộ đệm ==="
grep "DELIVERING FROM BUFFER" logs/process_0.log | head -5
```

**Lời Thoại:**
> "Hãy chú ý cách các tin nhắn đang được gửi từ bộ đệm sau đó. Hệ thống là tham lam - bất cứ khi nào chúng ta nhận được một tin nhắn mới, chúng ta sẽ cố gắng ngay lập tức gửi bất kỳ tin nhắn nào được lưu vào bộ đệm mà bây giờ đã có các phụ thuộc được thỏa mãn.
>
> Điều này đảm bảo:
> 1. Không có tin nhắn nào được gửi trước các phụ thuộc của nó
> 2. Không có lưu trữ đệm không cần thiết - các tin nhắn được gửi sớm nhất có thể
> 3. Ứng dụng thấy một luồng sự kiện nhất quán về nhân quả"

---

## Phần 7: Thống Kê Cuối Cùng (1 phút)

**Lời Thoại:**
> "Hãy kiểm tra các thống kê cuối cùng để xác minh tính chính xác. Tất cả các quy trình đều phải hoàn tất thành công."

**Trên Màn Hình:**
```bash
echo "=== Xác Minh Gửi Tin Nhắn ==="
echo ""
for i in {0..14}; do
  sent=$(grep "SENT" logs/process_$i.log | wc -l)
  delivered=$(grep "DELIVERED" logs/process_$i.log | wc -l)
  buffered=$(grep "BUFFERED" logs/process_$i.log | wc -l)
  printf "P%-2d: SENT=%4d | DELIVERED=%4d | BUFFERED NOW=%4d\n" \
    $i $sent $delivered $buffered
done

echo ""
echo "=== Tổng Hệ Thống ==="
echo "Tổng số tin nhắn được gửi: $(grep 'SENT' logs/*.log | wc -l)"
echo "Tổng số tin nhắn được gửi: $(grep 'DELIVERED' logs/*.log | wc -l)"
echo "Tổng số tin nhắn được lưu vào bộ đệm trong quá trình chạy: $(grep 'BUFFERED' logs/*.log | wc -l)"
echo "Tin nhắn vẫn được lưu vào bộ đệm ở cuối: $(grep 'Buffer size: [1-9]' logs/*.log | wc -l) quy trình có tin nhắn được lưu vào bộ đệm"
```

**Lời Thoại:**
> "Hoàn hảo! Chúng ta có thể thấy rằng tất cả các tin nhắn đã được gửi thành công. Thuật toán đã duy trì đúng thứ tự nhân quả trong suốt quá trình thực thi.
>
> Những quan sát chính:
> 1. Mỗi tin nhắn được gửi cũng được gửi
> 2. Bộ đệm cuối cùng trống - tất cả các tin nhắn cuối cùng được gửi
> 3. Các tin nhắn được lưu vào bộ đệm tạm thời, nhưng được phát hành khi các phụ thuộc đến
> 4. Hệ thống vẫn nhất quán trong suốt"

---

## Phần 8: Hướng Dẫn Mã (1-2 phút)

**Lời Thoại:**
> "Hãy nhanh chóng kiểm tra triển khai thuật toán chính."

**Trên Màn Hình:**
```bash
# Hiển thị logic CanDeliver
less pkg/vectorclock/vectorclock.go
# Chuyển đến hàm CanDeliver (xung quanh dòng 115)
```

**Lời Thoại (chỉ vào màn hình):**
> "Đây là logic quyết định cốt lõi. Khi một tin nhắn đến với thông tin vector của nó, chúng ta kiểm tra: tin nhắn này có mục nhập cho chúng ta không? Nếu có, tất cả các phụ thuộc của nó có được thỏa mãn không?
>
> Dòng chính: 'if entryForMe.Timestamp[j] > localTime[j]', chúng ta có một phụ thuộc chưa được thỏa mãn.
>
> Nếu tất cả các phụ thuộc được thỏa mãn, chúng ta có thể gửi ngay lập tức. Nếu không, chúng ta lưu vào bộ đệm và chờ đợi."

**Hiển Thị:**
- Hàm CanDeliver
- Logic kiểm tra phụ thuộc
- Giá trị trả về (đúng/sai với lý do)

---

## Phần 9: Tóm Tắt Thuật Toán (1 phút)

**Lời Thoại:**
> "Hãy tóm tắt những gì chúng ta đã trình bày:
>
> **Thuật Toán SES:**
> 1. Mỗi quy trình duy trì một vector clock (một bộ đếm cho mỗi quy trình)
> 2. Trước khi gửi một tin nhắn, chúng ta bao gồm vector clock hiện tại của chúng ta và lịch sử gửi gần đây
> 3. Khi nhận một tin nhắn, chúng ta kiểm tra xem các phụ thuộc của nó có được thỏa mãn không
> 4. Nếu các phụ thuộc được thỏa mãn, hãy gửi ngay lập tức và cố gắng gửi bất kỳ tin nhắn nào được lưu vào bộ đệm
> 5. Nếu không, hãy lưu tin nhắn vào bộ đệm và thử lại sau
>
> **Những Lợi Ích Chính:**
> - Đảm bảo thứ tự nhân quả của các tin nhắn
> - Chi phí mạng tối thiểu (chỉ gắn thêm thông tin cần thiết)
> - Không cần bộ tuần tự trung tâm
> - Mở rộng tốt cho nhiều quy trình
>
> **Kết Quả Của Chúng Tôi:**
> - Phối hợp thành công 15 quy trình đồng thời
> - Gửi hơn 20.000 tin nhắn với nhân quả hoàn hảo
> - Bộ đệm được xử lý hoàn toàn - không có tin nhắn bị mất hoặc mắc kẹt"

**Trên Màn Hình:**
Hiển thị phần thuật toán README

---

## Phần 10: Kết Luận (30 giây)

**Lời Thoại:**
> "Cảm ơn bạn đã xem bản demo về triển khai thuật toán SES. Dự án này cho thấy cách các hệ thống phân tán có thể duy trì tính nhất quán và nhân quả đồng thời giảm thiểu chi phí giao tiếp.
>
> Mã được ghi chép tốt và có sẵn để xem xét. Để biết thêm chi tiết, vui lòng tham khảo các file README.md, SETUP_GUIDE.md và DESIGN.md được bao gồm trong dự án.
>
> Có câu hỏi không?"

**Trên Màn Hình:**
- Hiển thị kho GitHub (nếu công khai)
- Liệt kê tất cả các file tài liệu
- Màn hình thống kê cuối cùng

---

## Mẹo Quay Phim

### Cài Đặt Kỹ Thuật
- Sử dụng OBS hoặc ScreenFlow với độ phân giải 1080p
- Tốc độ khung hình: tối thiểu 30 fps
- Microphone: Sử dụng microphone bên ngoài để có chất lượng tốt hơn
- Nền: Không gian sạch sẽ, yên tĩnh (tiếng ồn tối thiểu)

### Trong Quá Trình Quay
- Nói rõ ràng với tốc độ vừa phải
- Tạm dừng ngắn sau mỗi phần để chỉnh sửa
- Nếu bạn mắc lỗi, hãy tạm dừng và bắt đầu lại phần đó
- Làm nổi bật các phần quan trọng bằng cách di chuyển con trỏ lên chúng
- Cho phép tạm dừng 2-3 giây giữa các phần chính

### Chỉnh Sửa Sau Sản Xuất
- Cắt bỏ các tạm dừng và lỗi
- Thêm tiêu đề phần / chuyển tiếp
- Cân nhắc thêm các phủ định văn bản cho các khái niệm chính
- Phóng to mã để rõ ràng
- Chiều dài cuối cùng: 5-10 phút (chặt chẽ, chuyên nghiệp)

### Siêu Dữ Liệu Video
- Tiêu đề: "Triển Khai Thuật Toán SES - Sắp Xếp Tin Nhắn Phân Tán"
- Mô Tả: Bao gồm liên kết đến GitHub / file dự án, tóm tắt ngắn
- Thẻ: hệ-thống-phân-tán, vector-clocks, sắp-xếp-tin-nhắn, Go

---

## Kịch Bản Demo Thay Thế

### Demo Nhanh (3 phút)
- Bỏ qua hướng dẫn mã chi tiết
- Tập trung vào: Vấn Đề → Giải Pháp → Kết Quả
- Chỉ hiển thị các đoạn nhật ký chính

### Demo Mở Rộng (15 phút)
- Bao gồm hướng dẫn mã chi tiết về tất cả các thành phần
- Giải thích từng bước thuật toán với các ví dụ
- Mã hóa trực tiếp: sửa đổi và biên dịch lại (ví dụ: hiển thị những gì xảy ra mà không có vector clocks)

### Demo Tương Tác
- Chạy các quy trình một cách tương tác bằng cách sử dụng CLI
- Trình bày các lệnh 's' (gửi), 'i' (thông tin), 'b' (được lưu vào bộ đệm), 'v' (vector)
- Hiển thị cách bạn có thể giám sát các quy trình riêng lẻ

---

## Kết Quả Demo Dự Kiến (Để Tham Khảo)

Khi bạn chạy bản demo, bạn sẽ thấy:
- Xây dựng hoàn tất trong < 5 giây
- Các quy trình bắt đầu và xuất ra "Quy trình đã bắt đầu thành công"
- Các tin nhắn SENT xuất hiện với khoảng ~100 mỗi phút (có thể cấu hình)
- Các tin nhắn RECEIVED xuất hiện ngay sau đó
- Các tin nhắn DELIVERED xuất hiện sau khi các phụ thuộc được thỏa mãn
- Các tin nhắn BUFFERED thỉnh thoảng (quan trọng để trình bày!)
- Các tin nhắn DELIVERING FROM BUFFER (cho thấy thuật toán đang hoạt động)
- Các thống kê cuối cùng hiển thị tất cả các tin nhắn được gửi
- Tổng thời gian chạy: 45-75 giây

---

## Khắc Phục Sự Cố Khi Quay Phim

**Vấn Đề: Quá nhiều tin nhắn, màn hình quá tải**
- Giải Pháp: Sử dụng Grep để hiển thị các sự kiện cụ thể: `tail -f logs/process_0.log | grep DELIVERED`

**Vấn Đề: Các quy trình hoàn tất quá nhanh**
- Giải Pháp: Sửa đổi cấu hình để tăng `messages_per_process` hoặc giảm `messages_per_minute`

**Vấn Đề: Xung đột cổng**
- Giải Pháp: Giết bất kỳ quy trình nào bị mắc kẹt: `pkill -9 ses.exe`

**Vấn Đề: Vấn đề âm thanh**
- Giải Pháp: Quay lại phần đó riêng biệt và chỉnh sửa sau sản xuất

---

## Danh Sách Kiểm Tra Cuối Cùng Trước Gửi

- [ ] Video dài 5-10 phút
- [ ] Âm thanh rõ ràng và được kể chuyện chuyên nghiệp
- [ ] Tất cả các phần được bao gồm (giới thiệu, thiết lập, thực thi, kết quả, tóm tắt)
- [ ] Mã hiển thị và dễ đọc
- [ ] Các khái niệm chính được giải thích rõ ràng
- [ ] Demo thành công cho thấy lưu trữ đệm và gửi
- [ ] Các thống kê cuối cùng xác minh tính chính xác
- [ ] Không có tiếng ồn hoặc sao lãng nền
- [ ] Định dạng video: MP4, WebM hoặc liên kết YouTube
- [ ] File phụ đề / chú thích được bao gồm (tùy chọn nhưng được khuyến cáo)

---

**Phiên Bản Kịch Bản**: 1.0
**Thời Gian Quay Ước Tính**: 45 phút (bao gồm nhiều lần quay)
**Chiều Dài Video Cuối Cùng**: 5-10 phút
**Cập Nhật Lần Cuối**: 2025-11-27
